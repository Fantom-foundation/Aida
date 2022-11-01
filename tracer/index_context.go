package tracer

import (
	"log"
)

// IndexContext encapsulates all index data strutures.
type IndexContext struct {
	BlockIndex *BlockIndex // block-index
}

// NewIndexContext creates a new index context.
func NewIndexContext() *IndexContext {
	return &IndexContext{
		BlockIndex: NewBlockIndex()}
}

// ReadIndexContext reads a new index context from file(s).
func ReadIndexContext() *IndexContext {
	ctx := NewIndexContext()
	err := ctx.BlockIndex.Read(TraceDir + "block-index.dat")
	if err != nil {
		log.Fatalf("Cannot read block index. Error: %v", err)
	}
	return ctx
}

// Write the index context to file(s).
func (ctx *IndexContext) Write() {
	err := ctx.BlockIndex.Write(TraceDir + "block-index.dat")
	if err != nil {
		log.Fatalf("Cannot write block index. Error: %v", err)
	}
}

// AddBlock adds a block to the block index.
func (ctx *IndexContext) AddBlock(block uint64, fpos int64) {
	err := ctx.BlockIndex.Add(block, fpos)
	if err != nil {
		log.Fatalf("Adding block to block-index failed. Error: %v", err)
	}
}

// GetBlock gets a block from the block index.
func (ctx *IndexContext) GetBlock(block uint64) int64 {
	fpos, err := ctx.BlockIndex.Get(block)
	if err != nil {
		log.Fatalf("Getting block from block-index failed. Error: %v", err)
	}
	return fpos
}

// ExistsBlock checks whether block exists in block index.
func (ctx *IndexContext) ExistsBlock(block uint64) bool {
	exists, err := ctx.BlockIndex.Exists(block)
	if err != nil {
		log.Fatalf("Checking block from block-index failed. Error: %v", err)
	}
	return exists
}
