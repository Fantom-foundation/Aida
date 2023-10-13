package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type TestSyncPeriodEmitter[T any] struct {
	extension.NilExtension[T]
	config     *utils.Config
	syncPeriod uint64
}

// MakeTestSyncPeriodEmitter creates an executor.Extension handling SyncPeriod beginnings and ends.
func MakeTestSyncPeriodEmitter[T any](config *utils.Config) executor.Extension[T] {
	if config.SyncPeriodLength <= 0 {
		log := logger.NewLogger(config.LogLevel, "Progress-Reporter")
		log.Warning("SyncPeriodLength was set in config to 0; SyncPeriodEmitter disabled")
		return extension.NilExtension[T]{}
	}

	return &TestSyncPeriodEmitter[T]{config: config, syncPeriod: 0}
}

// PreRun checks whether syncPeriodLength isn't invalid
func (l *TestSyncPeriodEmitter[T]) PreRun(state executor.State[T], context *executor.Context) error {
	// initiate a sync period
	l.syncPeriod = uint64(state.Block) / l.config.SyncPeriodLength
	context.State.BeginSyncPeriod(l.syncPeriod)

	return nil
}

// PreBlock calculates current sync period and then invokes necessary state operations.
func (l *TestSyncPeriodEmitter[T]) PreBlock(state executor.State[T], context *executor.Context) error {
	// calculate the syncPeriod for given block
	newSyncPeriod := uint64(state.Block) / l.config.SyncPeriodLength

	// loop because multiple periods could have been empty
	for l.syncPeriod < newSyncPeriod {
		context.State.EndSyncPeriod()
		l.syncPeriod++
		context.State.BeginSyncPeriod(l.syncPeriod)
	}

	return nil
}

func (l *TestSyncPeriodEmitter[T]) PostRun(_ executor.State[T], context *executor.Context, _ error) error {
	context.State.EndSyncPeriod()
	return nil
}
