package blockprocessor

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/utils"
)

// DbManagerExtension mananges state-db directory
type DbManagerExtension struct {
	ProcessorExtensions
}

func NewDbManagerExtension() *DbManagerExtension {
	return &DbManagerExtension{}
}

func (la *DbManagerExtension) Init(bp *BlockProcessor) error {
	return nil
}

func (la *DbManagerExtension) PostPrepare(bp *BlockProcessor) error {
	return nil
}

func (la *DbManagerExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

// PostProcessing writes state-db info file
func (la *DbManagerExtension) PostProcessing(bp *BlockProcessor) error {
	if bp.cfg.KeepDb {
		rootHash, _ := bp.db.Commit(true)
		if err := utils.WriteStateDbInfo(bp.stateDbDir, bp.cfg, bp.block, rootHash); err != nil {
			return fmt.Errorf("failed to create state-db info file; %v", err)
		}
	}
	return nil
}

// Exit rename or remove state-db directory depending on the flags.
func (la *DbManagerExtension) Exit(bp *BlockProcessor) error {
	if bp.cfg.KeepDb {
		newName := utils.RenameTempStateDBDirectory(bp.cfg, bp.stateDbDir, bp.block)
		bp.log.Noticef("State-db directory: %v", newName)
	} else {
		bp.log.Warningf("--keep-db is not used. Directory %v with DB will be removed at the end of this run.", bp.stateDbDir)
		return os.RemoveAll(bp.stateDbDir)
	}
	return nil
}
