package tracer

import (
	"bufio"
	"io"
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/dsnet/compress/bzip2"
)

// TraceIterator data structure for storing state of a trace iterator
type TraceIterator struct {
	firstBlock   uint64              // first block to process
	currentBlock uint64              // current block to process
	file         *os.File            // trace file
	reader       *bufio.Reader       // read buffer
	zreader      *bzip2.Reader       // compressed stream
	currentOp    operation.Operation // current state operation
	currentFile  int                 // current file index
	fileList     []string            // list of trace file names
}

// NewTraceIterator creates a new trace iterator.
func NewTraceIterator(files []string, first uint64) *TraceIterator {
	// create new iterator object
	ti := new(TraceIterator)
	ti.firstBlock = first
	ti.currentBlock = 0
	ti.currentFile = 0
	ti.fileList = files
	// open trace file,read buffer, and gzip stream
	ti.OpenCurrentTraceFile()
	return ti
}

// OpenCurrentTraceFile reads a trace file at current file index
func (ti *TraceIterator) OpenCurrentTraceFile() {
	var err error
	if ti.file, err = os.Open(ti.fileList[ti.currentFile]); err != nil {
		log.Fatalf("cannot open trace file; %v", err)
	}
	ti.zreader, err = bzip2.NewReader(ti.file, &bzip2.ReaderConfig{})
	if err != nil {
		log.Fatalf("cannot open bzip stream; %v", err)
	}
	ti.reader = bufio.NewReaderSize(ti.zreader, 65536*256) // set buffer to 1MB
	//skip header
	var header [8]byte
	if _, err := io.ReadFull(ti.reader, header[:]); err != nil {
		log.Fatalf("fail to read file  header; %v", err)
	}
}

// Next loads the next operation from the trace file.
func (ti *TraceIterator) Next() bool {
	var err error
	for {
		// read next operation
		if ti.currentOp, err = operation.Read(ti.reader); err != nil {
			if err == io.EOF {
				ti.currentFile++
				if ti.currentFile < len(ti.fileList) {
					ti.Release()
					ti.OpenCurrentTraceFile()
					continue
				}
			} else if err != nil {
				log.Fatal(err)
			}
			return false
		}

		// update current block number
		if ti.currentOp.GetId() == operation.BeginBlockID {
			bb, ok := ti.currentOp.(*operation.BeginBlock)
			if !ok {
				log.Fatalf("Downcasting basic-block failed.")
			}
			ti.currentBlock = bb.BlockNumber
		}

		// break out loop if first block surpassed
		if ti.currentBlock >= ti.firstBlock {
			return true
		}
	}
}

// Value retrieves the current state operation of the trace file.
func (ti *TraceIterator) Value() operation.Operation {
	return ti.currentOp
}

// Release the storage trace iterator.
func (ti *TraceIterator) Release() {
	if err := ti.zreader.Close(); err != nil {
		log.Fatalf("Cannot close compressed stream. Error: %v", err)
	}
	if err := ti.file.Close(); err != nil {
		log.Fatalf("Cannot close trace file. Error: %v", err)
	}
}
