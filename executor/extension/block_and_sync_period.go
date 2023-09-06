package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

type blockManager struct {
	NilExtension
	config       *utils.Config
	syncPeriod   uint64
	isFirstBlock bool
}

// MakeBlockBeginner creates a executor.Extension call beginBlock and handles SyncPeriod beginning and end.
func MakeBlockBeginner(config *utils.Config) executor.Extension {
	return &blockManager{config: config, syncPeriod: 0, isFirstBlock: true}
}

// PreBlock first needs to calculate current sync period and then invokes necessary state operations.
func (l *blockManager) PreBlock(state executor.State) error {
	// only when first block number is known then syncPeriod can be calculated - therefore this can't be done in preRun
	if l.isFirstBlock {
		// initiate a sync period
		l.syncPeriod = uint64(state.Block) / l.config.SyncPeriodLength
		state.State.BeginSyncPeriod(l.syncPeriod)
		l.isFirstBlock = false
	}

	// calculate the syncPeriod for given block
	newSyncPeriod := uint64(state.Block) / l.config.SyncPeriodLength

	// loop because multiple empty periods could have been empty
	for l.syncPeriod < newSyncPeriod {
		state.State.EndSyncPeriod()
		l.syncPeriod++
		state.State.BeginSyncPeriod(l.syncPeriod)
	}

	state.State.BeginBlock(uint64(state.Block))
	return nil
}

// PostBlock
func (l *blockManager) PostBlock(state executor.State) error {
	state.State.EndBlock()
	return nil
}

func (l *blockManager) PostRun(state executor.State, _ error) error {
	// end syncPeriod if at least one was started
	if !l.isFirstBlock {
		state.State.EndSyncPeriod()
	}

	return nil
}
