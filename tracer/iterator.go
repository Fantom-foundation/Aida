// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package tracer

import (
	"io"
	"log"

	"github.com/Fantom-foundation/Aida/tracer/operation"
)

// TraceIterator data structure for storing state of a trace iterator
type TraceIterator struct {
	firstBlock     uint64              // first block to process
	currentBlock   uint64              // current block to process
	currentOp      operation.Operation // current state operation
	currentFileIdx int                 // current file index
	fileList       []string            // list of trace file names
	tf             *TraceFile          // trace file object
}

// NewTraceIterator creates a new trace iterator.
func NewTraceIterator(files []string, first uint64) *TraceIterator {
	// create new iterator object
	ti := new(TraceIterator)
	ti.firstBlock = first
	ti.currentBlock = 0
	ti.currentFileIdx = 0
	ti.fileList = files
	// open trace file,read buffer, and gzip stream
	ti.OpenCurrentTraceFile()
	return ti
}

// OpenCurrentTraceFile reads a trace file at current file index
func (ti *TraceIterator) OpenCurrentTraceFile() {
	var err error
	if ti.tf, err = NewTraceFile(ti.fileList[ti.currentFileIdx]); err != nil {
		log.Fatalf("cannot open trace file; %v", err)
	}
}

// Next loads the next operation from the trace file.
func (ti *TraceIterator) Next() bool {
	var err error
	for {
		// read next operation
		if ti.currentOp, err = operation.Read(ti.tf.reader); err != nil {
			if err == io.EOF {
				ti.currentFileIdx++
				// file index in range, load new file then continue
				if ti.currentFileIdx < len(ti.fileList) {
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
	if err := ti.tf.Release(); err != nil {
		log.Fatal(err)
	}
}
