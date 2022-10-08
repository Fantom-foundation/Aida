package tracer

// IndexContext encapsulates index data strutures.
type IndexContext struct {
	BlockIndex *BlockIndex
}

// Create a new index context.
func NewIndexContext() *IndexContext {
	return &IndexContext{
		BlockIndex: NewBlockIndex()}
}

// Read a new index context from files.
func ReadIndexContext() *IndexContext {
	ctx := NewIndexContext()
	err := ctx.BlockIndex.Read(TraceDir + "block-index.dat")
	if err != nil {
		log.Fatalf("Cannot read block index. Error: %v", err)
	}
	return ctx
}

// Write block index
func (ctx *IndexContext) Write() {
	err := ctx.BlockIndex.Write(TraceDir + "block-index.dat")
	if err != nil {
		log.Fatalf("Cannot write block index. Error: %v", err)
	}
}
