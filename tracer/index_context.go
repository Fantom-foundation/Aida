package tracer

// IndexContext encapsulates all index data strutures.
type IndexContext struct {
	BlockIndex *BlockIndex
}

// Create a new index context.
func NewIndexContext() *IndexContext {
	return &IndexContext{
		BlockIndex: NewBlockIndex()}
}

// Read a new index context from files.
// TODO: Error handling
func ReadIndexContext() *IndexContext {
	ctx := NewIndexContext()
	ctx.BlockIndex.Read("block-index.dat")
	return ctx
}

// Write block index
// TODO: Error handling
func (ctx *IndexContext) Write() {
	ctx.BlockIndex.Write("block-index.dat")
}
