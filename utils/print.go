package utils

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Printer is a utility class to output data from the system
type Printer interface {
	Print() error
	Close()
}

type Printers struct {
	printers []Printer
}

func (ps *Printers) Print() {
	for _, p := range ps.printers {
		p.Print()
	}
}

func (ps *Printers) Close() {
	for _, p := range ps.printers {
		p.Close()
	}
}

func NewPrinters() *Printers {
	return &Printers{[]Printer{}}
}

func (ps *Printers) AddPrinter(p Printer) *Printers {
	ps.printers = append(ps.printers, p)
	return ps
}

// PrinterToWriter writes to any io.Writer
// Wrap f, returns a string to be printed
type PrinterToWriter struct {
	w io.Writer
	f func() string
}

func (p *PrinterToWriter) Print() error {
	fmt.Fprintln(p.w, p.f())
	return nil
}

func (p *PrinterToWriter) Close() {
	return
}

func NewPrinterToWriter(w io.Writer, f func() string) *PrinterToWriter {
	return &PrinterToWriter{w, f}
}

func NewPrinterToConsole(f func() string) *PrinterToWriter {
	return &PrinterToWriter{os.Stdout, f}
}

func (ps *Printers) AddPrinterToWriter(w io.Writer, f func() string) *Printers {
	return ps.AddPrinter(NewPrinterToWriter(w, f))
}

func (ps *Printers) AddPrinterToConsole(isDisabled bool, f func() string) *Printers {
	if isDisabled {
		return ps
	}
	return ps.AddPrinter(NewPrinterToConsole(f))
}

// PrinterToFile writes to a File
// Wrap f, returns a string to be printed
type PrinterToFile struct {
	filepath string
	f        func() string
}

func (p *PrinterToFile) Print() error {
	file, err := os.OpenFile(p.filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to print to file %s; %v", p.filepath, err)
	}

	defer file.Close()
	file.WriteString(p.f())
	return nil
}

func (p *PrinterToFile) Close() {
	return
}

func NewPrinterToFile(filepath string, f func() string) *PrinterToFile {
	return &PrinterToFile{filepath, f}
}

func (ps *Printers) AddPrinterToFile(filepath string, f func() string) *Printers {
	if filepath != "" {
		ps.AddPrinter(NewPrinterToFile(filepath, f))
	}
	return ps
}

// PrinterToDb writes by inserting rows into DB
// Wrap f, returns an array of values to be inserted
type PrinterToDb struct {
	db     *sql.DB
	insert string
	f      func() [][]any
}

func (p *PrinterToDb) Print() error {
	// Transaction is used to improve efficiency over bulk insert
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("unable to begin a transaction; %v", err)
	}

	stmt, err := p.db.Prepare(p.insert)
	if err != nil {
		return fmt.Errorf("unable to prepare statement %s; %v", p.insert, err)
	}

	values := p.f()
	for _, value := range values {
		_, err = tx.Stmt(stmt).Exec(value...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	defer stmt.Close() // Stmt to be open/close each time a transaction happens
	return tx.Commit()
}

func (p *PrinterToDb) Close() {
	p.db.Close()
}

func NewPrinterToSqlite3(conn string, create string, insert string, f func() [][]any) (*PrinterToDb, error) {
	var err error

	db, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection to sqlite3 %s; %v", conn, err)
	}

	_, err = db.Exec(create)
	if err != nil {
		return nil, fmt.Errorf("failed to create/replace table on %s; %v", conn, err)
	}

	db.Exec("PRAGMA synchronous = OFF")     // so that insert does not block
	db.Exec("PRAGMA journal_mode = MEMORY") // improve efficiency - no intermediate write to file

	return &PrinterToDb{db, insert, f}, nil
}

func (ps *Printers) AddPrinterToSqlite3(conn string, create string, insert string, f func() [][]any) *Printers {
	if conn != "" {
		p, err := NewPrinterToSqlite3(conn, create, insert, f)
		if err != nil {
			return ps
		}
		return ps.AddPrinter(p)
	}
	return ps
}

// Bufferize split PrintToDB into 2 printers: 1. print to buffer 2. flush buffer to DB
func (p *PrinterToDb) Bufferize(capacity int) (*PrinterToBuffer, *Flusher) {
	pb := &PrinterToBuffer{capacity, p.f, make([][]any, capacity), nil}
	flusher := &Flusher{p, pb}
	pb.flusher = flusher
	return pb, flusher
}

type PrinterToBuffer struct {
	capacity int
	f        func() [][]any
	buffer   [][]any
	flusher  *Flusher
}

func (p *PrinterToBuffer) Print() error {
	p.buffer = append(p.buffer, p.f()...)
	if len(p.buffer) >= p.capacity {
		return p.flusher.Print()
	}

	return nil
}

func (p *PrinterToBuffer) Close() {
	return
}

func (p *PrinterToBuffer) Reset() {
	p.buffer = p.buffer[:0]
}

func (p *PrinterToBuffer) Length() int {
	return len(p.buffer)
}

type Flusher struct {
	og *PrinterToDb     // needs to know how the original printer prints
	bf *PrinterToBuffer // needs to access the buffer
}

func (p *Flusher) Print() error {
	p.og.f = func() [][]any { return p.bf.buffer }

	defer p.bf.Reset() // clear buffer here
	return p.og.Print()
}

func (p *Flusher) Close() {
	p.og.Close()
	p.bf.Close()
}
