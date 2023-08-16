package blockprocessor

import (
	"github.com/Fantom-foundation/Aida/state"
)

type VMAdbExtension struct {
	ProcessorExtensions
	db state.StateDB
}

func NewVMAdbExtension() *VMAdbExtension {
	return &VMAdbExtension{}
}

func (ext *VMAdbExtension) Init(bp *BlockProcessor) error {

	return nil
}

// PostPrepare validates the world-state after preparing/priming db
func (ext *VMAdbExtension) PostPrepare(bp *BlockProcessor) error {
	var err error

	// we need to save reference to StateDb itself in order to extract ArchiveDb for each block
	ext.db = bp.db

	// todo what to do when we start at block 0?
	bp.db, err = bp.db.GetArchiveState(bp.cfg.First - 1)
	if err != nil {
		return err
	}

	bp.db.BeginBlock(bp.cfg.First)

	return nil
}

func (ext *VMAdbExtension) PostBlock(bp *BlockProcessor) error {
	var err error

	bp.db, err = ext.db.GetArchiveState(bp.currentTx.Block - 1)
	if err != nil {
		return err
	}

	// Mark the beginning of a new block
	bp.block = bp.currentTx.Block
	bp.db.BeginBlock(bp.block)

	return nil
}

func (ext *VMAdbExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

// PostProcessing checks the world-state after processing has completed
func (ext *VMAdbExtension) PostProcessing(bp *BlockProcessor) error {

	return nil
}

func (ext *VMAdbExtension) Exit(bp *BlockProcessor) error {
	return nil
}
