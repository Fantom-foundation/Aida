package runvm

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
)

type ValidationAction struct {
	ProcessorActions
}

func NewValidationAction() *ValidationAction {
	return &ValidationAction{}
}

func (la *ValidationAction) Init(bp *BlockProcessor) error {
	return nil
}

// PostPrepare validates the worldstate after preparing/priming db
func (la *ValidationAction) PostPrepare(bp *BlockProcessor) error {
	if bp.cfg.ValidateWorldState {
		bp.log.Notice("Validate primed world-state\n")
		ws, err := utils.GenerateWorldStateFromUpdateDB(bp.cfg, bp.cfg.First-1)
		if err != nil {
			return fmt.Errorf("failed generating worldstate. %v", err)
		}
		if err := utils.DeleteDestroyedAccountsFromWorldState(ws, bp.cfg, bp.cfg.First-1); err != nil {
			return fmt.Errorf("failed to remove deleted accoount from the world state. %v", err)
		}
		if err := utils.ValidateStateDB(ws, bp.db, false); err != nil {
			return fmt.Errorf("pre: World state is not contained in the stateDB. %v", err)
		}
	}
	return nil
}

func (la *ValidationAction) PostTransaction(bp *BlockProcessor) error {
	return nil
}

// PostProcessing checks the world-state after processing has completed
func (la *ValidationAction) PostProcessing(bp *BlockProcessor) error {
	if bp.cfg.ValidateWorldState {
		bp.log.Notice("Validate final world-state\n")
		ws, err := utils.GenerateWorldStateFromUpdateDB(bp.cfg, bp.cfg.Last)
		if err != nil {
			return err
		}
		if err := utils.DeleteDestroyedAccountsFromWorldState(ws, bp.cfg, bp.cfg.Last); err != nil {
			return fmt.Errorf("Failed to remove deleted accoount from the world state. %v", err)
		}
		if err := utils.ValidateStateDB(ws, bp.db, false); err != nil {
			return fmt.Errorf("World state is not contained in the stateDB. %v", err)
		}
	}
	return nil
}

func (la *ValidationAction) Exit(bp *BlockProcessor) error {
	return nil
}
