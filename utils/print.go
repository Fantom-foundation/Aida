package utils

import (
	"fmt"
	"io"
	"os"
	"log"
)

type Printer interface {
	Print() error
}

type PrintToWriter struct {
	w io.Writer
	f func() string
}

func (p *PrintToWriter) Print() error {
	fmt.Fprintln(p.w, f())
	return nil
}

func NewPrintToWriter (w io.Writer, f func() string) *PrintToWriter {
	return &PrintToWriter{w, f}
}

func NewPrintToConsole (f func() string) *PrintToWriter {
	return &PrintToWriter{os.Stdout, f}
}

func PrintToFile struct {
	filepath string
	f func() string
}

func (p *PrintToFile) Print() error {
	file, err := os.OpenFile(ps.csv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to print to file %s - %v", p.filepath, err)
	}
	defer file.Close()
	file.WriteString(f())
	return nil
}

func NewPrintToFile(filepath string, f func() string) *PrintToFile {
	return &PrintToFile(string, f)	
}
