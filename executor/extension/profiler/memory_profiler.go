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
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeMemoryProfiler creates an executor.Extension that records memory profiling data if enabled in the configuration.
func MakeMemoryProfiler[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.MemoryProfile == "" {
		return extension.NilExtension[T]{}
	}
	return &memoryProfiler[T]{cfg: cfg}
}

type memoryProfiler[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
}

func (p *memoryProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	return utils.StartMemoryProfile(p.cfg)
}
