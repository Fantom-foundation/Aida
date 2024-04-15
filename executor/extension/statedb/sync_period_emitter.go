// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package statedb

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type TestSyncPeriodEmitter[T any] struct {
	extension.NilExtension[T]
	cfg        *utils.Config
	syncPeriod uint64
}

// MakeTestSyncPeriodEmitter creates an executor.Extension handling SyncPeriod beginnings and ends.
func MakeTestSyncPeriodEmitter[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.SyncPeriodLength <= 0 {
		log := logger.NewLogger(cfg.LogLevel, "Progress-Reporter")
		log.Warning("SyncPeriodLength was set in cfg to 0; SyncPeriodEmitter disabled")
		return extension.NilExtension[T]{}
	}

	return &TestSyncPeriodEmitter[T]{cfg: cfg, syncPeriod: 0}
}

// PreRun checks whether syncPeriodLength isn't invalid
func (l *TestSyncPeriodEmitter[T]) PreRun(state executor.State[T], ctx *executor.Context) error {
	// initiate a sync period
	l.syncPeriod = uint64(state.Block) / l.cfg.SyncPeriodLength
	ctx.State.BeginSyncPeriod(l.syncPeriod)

	return nil
}

// PreBlock calculates current sync period and then invokes necessary state operations.
func (l *TestSyncPeriodEmitter[T]) PreBlock(state executor.State[T], ctx *executor.Context) error {
	// calculate the syncPeriod for given block
	newSyncPeriod := uint64(state.Block) / l.cfg.SyncPeriodLength

	// loop because multiple periods could have been empty
	for l.syncPeriod < newSyncPeriod {
		ctx.State.EndSyncPeriod()
		l.syncPeriod++
		ctx.State.BeginSyncPeriod(l.syncPeriod)
	}

	return nil
}

func (l *TestSyncPeriodEmitter[T]) PostRun(_ executor.State[T], ctx *executor.Context, _ error) error {
	ctx.State.EndSyncPeriod()
	return nil
}
