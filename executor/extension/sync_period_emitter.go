package extension

import (
	"errors"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

type SyncPeriodHook struct {
	NilExtension
	config       *utils.Config
	syncPeriod   uint64
	isFirstBlock bool
}

// MakeSyncPeriodHook creates an executor.Extension handling SyncPeriod beginnings and ends.
func MakeSyncPeriodHook(config *utils.Config) executor.Extension {
	return &SyncPeriodHook{config: config, syncPeriod: 0, isFirstBlock: true}
}

// PreBlock first needs to calculate current sync period and then invokes necessary state operations.
func (l *SyncPeriodHook) PreBlock(state executor.State) error {
	if l.config.SyncPeriodLength == 0 {
		return errors.New("syncPeriodLength from config can't be set to 0")
	}

	// only when first block number is known then syncPeriod can be calculated - therefore this can't be done in preRun
	if l.isFirstBlock {
		// initiate a sync period
		l.syncPeriod = uint64(state.Block) / l.config.SyncPeriodLength
		state.State.BeginSyncPeriod(l.syncPeriod)
		l.isFirstBlock = false
	}

	// calculate the syncPeriod for given block
	newSyncPeriod := uint64(state.Block) / l.config.SyncPeriodLength

	// loop because multiple periods could have been empty
	for l.syncPeriod < newSyncPeriod {
		state.State.EndSyncPeriod()
		l.syncPeriod++
		state.State.BeginSyncPeriod(l.syncPeriod)
	}

	return nil
}

func (l *SyncPeriodHook) PostRun(state executor.State, _ error) error {
	// end syncPeriod if at least one was started
	if !l.isFirstBlock {
		state.State.EndSyncPeriod()
	}

	return nil
}
