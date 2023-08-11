package blockprocessor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
)

type ValidationExtension struct {
	ProcessorExtensions
}

func NewValidationExtension() *ValidationExtension {
	return &ValidationExtension{}
}

func (ext *ValidationExtension) Init(bp *BlockProcessor) error {
	return nil
}

// PostPrepare validates the worldstate after preparing/priming db
func (ext *ValidationExtension) PostPrepare(bp *BlockProcessor) error {
	if bp.cfg.ValidateWorldState {
		bp.log.Notice("Validate primed world-state\n")
		ws, err := utils.GenerateWorldStateFromUpdateDB(bp.cfg, bp.cfg.First-1)
		if err != nil {
			return fmt.Errorf("failed generating worldstate. %v", err)
		}
		if err := utils.ValidateStateDB(ws, bp.db, false); err != nil {
			return fmt.Errorf("pre: World state is not contained in the stateDB. %v", err)
		}
	}
	return nil
}

func (ext *ValidationExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

// PostProcessing checks the world-state after processing has completed
func (ext *ValidationExtension) PostProcessing(bp *BlockProcessor) error {
	if bp.cfg.ValidateWorldState {
		bp.log.Notice("Validate final world-state\n")
		ws, err := utils.GenerateWorldStateFromUpdateDB(bp.cfg, bp.cfg.Last)
		if err != nil {
			return err
		}
		if err := utils.ValidateStateDB(ws, bp.db, false); err != nil {
			return fmt.Errorf("World state is not contained in the stateDB. %v", err)
		}
	}
	return nil
}

func (ext *ValidationExtension) Exit(bp *BlockProcessor) error {
	return nil
}
