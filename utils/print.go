package utils

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

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

type PrintToWriter struct {
	w io.Writer
	f func() string
}

func (p *PrintToWriter) Print() error {
	fmt.Fprintln(p.w, p.f())
	return nil
}

func (p *PrintToWriter) Close() {
	return
}

func NewPrintToWriter(w io.Writer, f func() string) *PrintToWriter {
	return &PrintToWriter{w, f}
}

func NewPrintToConsole(f func() string) *PrintToWriter {
	return &PrintToWriter{os.Stdout, f}
}

func (ps *Printers) AddPrintToWriter(w io.Writer, f func() string) *Printers {
	return ps.AddPrinter(NewPrintToWriter(w, f))
}

func (ps *Printers) AddPrintToConsole(isDisabled bool, f func() string) *Printers {
	if isDisabled {
		return ps
	}
	return ps.AddPrinter(NewPrintToConsole(f))
}

type PrintToFile struct {
	filepath string
	f        func() string
}

func (p *PrintToFile) Print() error {
	file, err := os.OpenFile(p.filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to print to file %s - %v", p.filepath, err)
	}
	defer file.Close()
	file.WriteString(p.f())
	return nil
}

func (p *PrintToFile) Close() {
	return
}

func NewPrintToFile(filepath string, f func() string) *PrintToFile {
	return &PrintToFile{filepath, f}
}

func (ps *Printers) AddPrintToFile(filepath string, f func() string) *Printers {
	if filepath != "" {
		ps.AddPrinter(NewPrintToFile(filepath, f))
	}
	return ps
}

type PrintToDb struct {
	db     *sql.DB
	insert string
	f      func() [][]any
}

func (p *PrintToDb) Print() error {
	stmt, err := p.db.Prepare(p.insert)

	if err != nil {
		return fmt.Errorf("unable to prepare statement, %s", p.insert)
	}

	defer stmt.Close()

	values := p.f()
	for _, value := range values {
		stmt.Exec(value...)
	}
	return nil
}

func (p *PrintToDb) Close() {
	p.db.Close()
}

func NewPrintToSqlite3(conn string, create string, insert string, f func() [][]any) (*PrintToDb, error) {
	var err error

	db, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, fmt.Errorf("unable to open connection to sqlite3 %s", conn)
	}

	_, err = db.Exec(create)
	if err != nil {
		return nil, fmt.Errorf("Could not confirm if table exists")
	}

	return &PrintToDb{db, insert, f}, err
}

func (ps *Printers) AddPrintToSqlite3(conn string, create string, insert string, f func() [][]any) *Printers {
	if conn != "" {
		p, err := NewPrintToSqlite3(conn, create, insert, f)
		if err != nil {
			return ps
		}
		return ps.AddPrinter(p)
	}
	return ps
}
