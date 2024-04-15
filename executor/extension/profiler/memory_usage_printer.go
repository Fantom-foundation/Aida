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

package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeMemoryUsagePrinter creates an executor.Extension that prints memory breakdown if enabled.
func MakeMemoryUsagePrinter[T any](cfg *utils.Config) executor.Extension[T] {
	if !cfg.MemoryBreakdown {
		return extension.NilExtension[T]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Memory-Usage-Printer")
	return makeMemoryUsagePrinter[T](cfg, log)
}

func makeMemoryUsagePrinter[T any](cfg *utils.Config, log logger.Logger) executor.Extension[T] {
	return &memoryUsagePrinter[T]{
		log: log,
		cfg: cfg,
	}
}

type memoryUsagePrinter[T any] struct {
	extension.NilExtension[T]
	log logger.Logger
	cfg *utils.Config
}

func (p *memoryUsagePrinter[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	if ctx.State != nil {
		utils.MemoryBreakdown(ctx.State, p.cfg, p.log)
	}
	return nil
}

func (p *memoryUsagePrinter[T]) PostRun(_ executor.State[T], ctx *executor.Context, _ error) error {
	if ctx.State != nil {
		utils.MemoryBreakdown(ctx.State, p.cfg, p.log)
	}
	return nil
}
