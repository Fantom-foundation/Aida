package utils

import (
	"fmt"
	"io"
	"os"
)

type Printer interface {
	Print() error
}

type Printers struct {
	printers []Printer
}

func (ps* Printers) Print() {
	for _, p := range ps.printers { p.Print() }
}

func NewPrinters() *Printers {
	return &Printers{[]Printer{}}
}

func (ps* Printers) AddPrinter(p Printer) *Printers {
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

func NewPrintToWriter(w io.Writer, f func() string) *PrintToWriter {
	return &PrintToWriter{w, f}
}

func NewPrintToConsole(f func() string) *PrintToWriter {
	return &PrintToWriter{os.Stdout, f}
}

func (ps* Printers) AddPrintToWriter(w io.Writer, f func() string) *Printers {
	return ps.AddPrinter(NewPrintToWriter(w, f))
}

func (ps* Printers) AddPrintToConsole(f func() string) *Printers {
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

func NewPrintToFile(filepath string, f func() string) *PrintToFile {
	return &PrintToFile{filepath, f}
}

func (ps* Printers) AddPrintToFile(filepath string, f func() string) *Printers {
	if filepath != "" {
		ps.AddPrinter(NewPrintToFile(filepath, f))
	}
	return ps
}

