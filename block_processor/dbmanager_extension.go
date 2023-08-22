package blockprocessor

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/utils"
)

// DbManagerExtension manages state-db directory
type DbManagerExtension struct {
	ProcessorExtensions
}

func NewDbManagerExtension() *DbManagerExtension {
	return &DbManagerExtension{}
}

func (ext *DbManagerExtension) Init(bp *BlockProcessor) error {
	if !bp.cfg.KeepDb {
		bp.log.Warningf("--keep-db was not used, the StateDb will be deleted after the run")
		return nil
	}

	return nil
}

func (ext *DbManagerExtension) PostPrepare(bp *BlockProcessor) error {
	return nil
}

func (ext *DbManagerExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

func (ext *DbManagerExtension) PostBlock(bp *BlockProcessor) error {
	return nil
}

// PostProcessing writes state-db info file
func (ext *DbManagerExtension) PostProcessing(bp *BlockProcessor) error {
	if !bp.cfg.KeepDb {
		return nil
	}

	rootHash, _ := bp.db.Commit(true)
	if err := utils.WriteStateDbInfo(bp.stateDbDir, bp.cfg, bp.block, rootHash); err != nil {
		return fmt.Errorf("failed to create state-db info file; %v", err)
	}

	return nil
}

// Exit rename or remove state-db directory depending on the flags.
func (ext *DbManagerExtension) Exit(bp *BlockProcessor) error {
	if bp.cfg.KeepDb {
		newName := utils.RenameTempStateDBDirectory(bp.cfg, bp.stateDbDir, bp.block)
		bp.log.Noticef("State-db directory: %v", newName)
		return nil
	}

	bp.log.Warningf("removing state-db %v", bp.stateDbDir)
	err := os.RemoveAll(bp.stateDbDir)
	if err != nil {
		return err
	}

	return nil
}
