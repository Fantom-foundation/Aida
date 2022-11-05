package tracer

import (
	"bufio"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/dsnet/compress/bzip2"
	"log"
	"os"
)

// TraceIterator data structure for storing state of a trace iterator
type TraceIterator struct {
	firstBlock   uint64              // last block to process
	currentBlock uint64              // current block to process
	lastBlock    uint64              // last block to process
	file         *os.File            // trace file
	reader       *bufio.Reader       // read buffer
	zreader      *bzip2.Reader       // compressed stream
	currentOp    operation.Operation // current state operation
}

// TraceDir is the directory of the trace files.
var TraceDir string = "./"

// NewTraceIterator creates a new trace iterator.
func NewTraceIterator(first uint64, last uint64) *TraceIterator {
	// create new iterator object
	ti := new(TraceIterator)
	ti.firstBlock = first
	ti.currentBlock = 0
	ti.lastBlock = last

	// open trace file,read buffer, and gzip stream
	var err error
	if ti.file, err = os.Open(TraceDir + "trace.dat"); err != nil {
		log.Fatalf("Cannot open trace file. Error: %v", err)
	}
	ti.zreader, err = bzip2.NewReader(ti.file, &bzip2.ReaderConfig{})
	if err != nil {
		log.Fatalf("Cannot open bzip stream. Error: %v", err)
	}
	ti.reader = bufio.NewReaderSize(ti.zreader, 65536*256) // set buffer to 1MB
	return ti
}

// Next loads the next operation from the trace file.
func (ti *TraceIterator) Next() bool {
	for {
		// read next operation
		if ti.currentOp = operation.Read(ti.reader); ti.currentOp == nil {
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
