package blockprocessor

type VMSdbExtension struct {
	ProcessorExtensions
}

func NewVMSdbExtension() *VMSdbExtension {
	return &VMSdbExtension{}
}

func (ext *VMSdbExtension) Init(bp *BlockProcessor) error {
	bp.cfg.CopySrcDb = true
	return nil
}

// PostPrepare validates the world-state after preparing/priming db
func (ext *VMSdbExtension) PostPrepare(bp *BlockProcessor) error {
	return nil
}

func (ext *VMSdbExtension) PostBlock(bp *BlockProcessor) error {

	bp.db.EndBlock()

	// switch to next sync-period if needed.
	// TODO: Revisit semantics - is this really necessary ????
	newSyncPeriod := bp.tx.Block / bp.cfg.SyncPeriodLength
	for bp.syncPeriod < newSyncPeriod {
		bp.db.EndSyncPeriod()
		bp.syncPeriod++
		bp.db.BeginSyncPeriod(bp.syncPeriod)
	}

	// Mark the beginning of a new block
	bp.block = bp.tx.Block
	bp.db.BeginBlock(bp.block)

	return nil
}

func (ext *VMSdbExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

// PostProcessing checks the world-state after processing has completed
func (ext *VMSdbExtension) PostProcessing(bp *BlockProcessor) error {

	return nil
}

func (ext *VMSdbExtension) Exit(bp *BlockProcessor) error {
	return nil
}
