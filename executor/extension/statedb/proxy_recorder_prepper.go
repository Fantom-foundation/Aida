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
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeProxyRecorderPrepper creates an extension which
// creates a temporary RecorderProxy before each txcontext
func MakeProxyRecorderPrepper[T any](cfg *utils.Config) executor.Extension[T] {
	return makeProxyRecorderPrepper[T](cfg)
}

func makeProxyRecorderPrepper[T any](cfg *utils.Config) *proxyRecorderPrepper[T] {
	return &proxyRecorderPrepper[T]{
		cfg: cfg,
	}
}

type proxyRecorderPrepper[T any] struct {
	extension.NilExtension[T]
	cfg        *utils.Config
	rCtx       *context.Record
	syncPeriod uint64
}

func (p *proxyRecorderPrepper[T]) PreRun(state executor.State[T], _ *executor.Context) error {
	var err error
	p.rCtx, err = context.NewRecord(p.cfg.TraceFile, p.cfg.First)
	if err != nil {
		return fmt.Errorf("cannot create record context; %v", err)
	}

	p.rCtx.Debug = p.cfg.Debug

	// write the first sync period
	p.syncPeriod = uint64(state.Block) / p.cfg.SyncPeriodLength
	operation.WriteOp(p.rCtx, operation.NewBeginSyncPeriod(p.syncPeriod))

	return nil
}

func (p *proxyRecorderPrepper[T]) PreBlock(state executor.State[T], ctx *executor.Context) error {
	// calculate the syncPeriod for given block
	newSyncPeriod := uint64(state.Block) / p.cfg.SyncPeriodLength

	// loop because multiple periods could have been empty
	for p.syncPeriod < newSyncPeriod {
		operation.WriteOp(p.rCtx, operation.NewEndSyncPeriod())
		p.syncPeriod++
		operation.WriteOp(p.rCtx, operation.NewBeginSyncPeriod(p.syncPeriod))
	}

	operation.WriteOp(p.rCtx, operation.NewBeginBlock(uint64(state.Block)))
	return nil
}

// PreTransaction checks whether ctx.State has not been overwritten by temporary prepper,
// if so it creates RecorderProxy.
func (p *proxyRecorderPrepper[T]) PreTransaction(_ executor.State[T], ctx *executor.Context) error {
	// if ctx.State has not been change, no need to slow down the app by creating new Proxy
	if _, ok := ctx.State.(*proxy.RecorderProxy); ok {
		return nil
	}

	ctx.State = proxy.NewRecorderProxy(ctx.State, p.rCtx)
	return nil
}

func (p *proxyRecorderPrepper[T]) PostBlock(executor.State[T], *executor.Context) error {
	operation.WriteOp(p.rCtx, operation.NewEndBlock())
	return nil
}

func (p *proxyRecorderPrepper[T]) PostRun(_ executor.State[T], ctx *executor.Context, err error) error {
	operation.WriteOp(p.rCtx, operation.NewEndSyncPeriod())
	p.rCtx.Close()
	return nil
}
