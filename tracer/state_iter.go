package tracer

import (
	"log"
	"os"
)

// Iterator data structure for storage traces
type TraceIterator struct {
	lastBlock uint64        // last block to process
	iCtx      *IndexContext // index context
	file      *os.File      // trace file
	currentOp Operation     // current state operation
}

// Output directory
var TraceDir string = "./"

// Create new trace iterator.
func NewTraceIterator(iCtx *IndexContext, first uint64, last uint64) *TraceIterator {
	p := new(TraceIterator)
	p.iCtx = iCtx
	p.lastBlock = last

	// TODO: Add trace directory to filename
	var err error
	p.file, err = os.OpenFile(TraceDir+"trace.dat", os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Cannot open trace file.")
	}
	// TODO: set file position to the first position using seek
	//_, err := file.Seek(iCtx.BlockIndex.Get(first), 0)
	//if err != nil {
	//		log.Fatalf("Cannot set position in trace file. Error: %v", err)
	//}
	return p
}

// Get next state operation from trace file.
func (ti *TraceIterator) Next() bool {
	// TODO: get file position for checking last
	//pos, err := file.Seek(0, 1)
	//if err != nil {
	//		log.Fatalf("Cannot set position in trace file. Error: %v", err)
	//}
	// if iCtx.BlockIndex.Exists(ti.last+1) {
	//  if pos >= iCtx.BlockIndex.Get(ti.last+1) {
	//       return false
	//  }
	// }
	ti.currentOp = ReadOperation(ti.file)
	return ti.currentOp != nil
}

// Retrieve current state operation of the iterator.
func (ti *TraceIterator) Value() Operation {
	return ti.currentOp
}

// Release the storage trace iterator.
func (ti *TraceIterator) Release() {
	err := ti.file.Close()
	if err != nil {
		log.Fatalf("Cannot close trace file")
	}
}
