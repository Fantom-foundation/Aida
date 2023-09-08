package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type TestSyncPeriodEmitter struct {
	NilExtension
	config     *utils.Config
	syncPeriod uint64
}

// MakeTestSyncPeriodEmitter creates an executor.Extension handling SyncPeriod beginnings and ends.
func MakeTestSyncPeriodEmitter(config *utils.Config) executor.Extension {
	if config.SyncPeriodLength <= 0 {
		log := logger.NewLogger(config.LogLevel, "Progress-Reporter")
		log.Warning("SyncPeriodLength was set in config to 0; SyncPeriodEmitter disabled")
		return NilExtension{}
	}

	return &TestSyncPeriodEmitter{config: config, syncPeriod: 0}
}

// PreRun checks whether syncPeriodLength isn't invalid
func (l *TestSyncPeriodEmitter) PreRun(state executor.State) error {
	// initiate a sync period
	l.syncPeriod = uint64(state.Block) / l.config.SyncPeriodLength
	state.State.BeginSyncPeriod(l.syncPeriod)

	return nil
}

// PreBlock calculates current sync period and then invokes necessary state operations.
func (l *TestSyncPeriodEmitter) PreBlock(state executor.State) error {
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

func (l *TestSyncPeriodEmitter) PostRun(state executor.State, _ error) error {
	state.State.EndSyncPeriod()
	return nil
}
