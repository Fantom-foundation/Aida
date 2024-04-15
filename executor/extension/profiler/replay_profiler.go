// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeReplayProfiler creates executor.Extension that prints profile statistics
func MakeReplayProfiler[T any](cfg *utils.Config, rCtx *context.Replay) executor.Extension[T] {
	if !cfg.Profile {
		return extension.NilExtension[T]{}
	}

	return &replayProfiler[T]{
		cfg:  cfg,
		rCtx: rCtx,
	}
}

type replayProfiler[T any] struct {
	extension.NilExtension[T]
	cfg  *utils.Config
	rCtx *context.Replay
}

func (p *replayProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.rCtx.Stats.FillLabels(operation.CreateIdLabelMap())
	if err := p.rCtx.Stats.PrintProfiling(p.cfg.First, p.cfg.Last); err != nil {
		return err
	}

	return nil
}
