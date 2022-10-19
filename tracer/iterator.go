package tracer

import (
	"log"
	"os"

	"github.com/Fantom-foundation/aida/tracer/operation"
)

// Iterator data structure for storage traces
type TraceIterator struct {
	lastBlock uint64              // last block to process
	iCtx      *IndexContext       // index context
	file      *os.File            // trace file
	currentOp operation.Operation // current state operation
}

// Output directory
var TraceDir string = "./"

// Create new trace iterator.
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

	return p
}

// Get next state operation from trace file.
func (ti *TraceIterator) Next() bool {
	// check whether we have processed all blocks in range
	if ti.iCtx.ExistsBlock(ti.lastBlock + 1) {
		// get file positions
		pos, err := ti.file.Seek(0, 1)
		if err != nil {
			log.Fatalf("Cannot get file position in trace file. Error: %v", err)
		}
		// end reached?
		if pos >= ti.iCtx.GetBlock(ti.lastBlock+1) {
			return false
		}
	}
	// read next state operation
	ti.currentOp = operation.Read(ti.file)
	return ti.currentOp != nil
}

// Retrieve current state operation of the iterator.
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
