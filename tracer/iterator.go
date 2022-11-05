package tracer

import (
	"bufio"
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/tracer/operation"
)

// TraceIterator data structure for storing state of a trace iterator
type TraceIterator struct {
	lastBlock uint64              // last block to process
	iCtx      *IndexContext       // index context
	file      *os.File            // trace file
	reader    *bufio.Reader       // buffered i/o file
	currentOp operation.Operation // current state operation
}

// TraceDir is the directory of the trace files.
var TraceDir string = "./"

// NewTraceIterator creates a new trace iterator.
func NewTraceIterator(iCtx *IndexContext, first uint64, last uint64) *TraceIterator {
	p := new(TraceIterator)
	p.iCtx = iCtx
	p.lastBlock = last

	var err error
	p.file, err = os.OpenFile(TraceDir+"trace.dat", os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Cannot open trace file. Error: %v", err)
	}

	// check whether first block exists
	if !iCtx.ExistsBlock(first) {
		log.Fatalf("First block does not exist. Error: %v", err)
	}

	// set file position
	_, err = p.file.Seek(iCtx.GetBlock(first), 0)
	if err != nil {
		log.Fatalf("Cannot set position in trace file. Error: %v", err)
	}

	// start buffering I/O
	p.reader = bufio.NewReaderSize(p.file, 65536*16)

	return p
}

// Next gets next operation from the trace file.
func (ti *TraceIterator) Next() bool {

	// read next state operation
	ti.currentOp = operation.Read(ti.reader)

	// check whether we have reached end of block range
	if ti.currentOp != nil {
		if ti.currentOp.GetId() == operation.BeginBlockID {
			beginBlock := ti.currentOp.(*operation.BeginBlock)
			if beginBlock == nil {
				log.Fatalf("Downcasting for BeginBlock failed.")
			}
			if beginBlock.BlockNumber > ti.lastBlock {
				return false
			}
		}
		return true
	} else {
		return false
	}
}

// Value retrieves teh current state operation of the trace file.
func (ti *TraceIterator) Value() operation.Operation {
	return ti.currentOp
}

// Release the storage trace iterator.
func (ti *TraceIterator) Release() {
	err := ti.file.Close()
	if err != nil {
		log.Fatalf("Cannot close trace file. Error: %v", err)
	}
}
