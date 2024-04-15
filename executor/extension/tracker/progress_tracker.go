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

package tracker

import (
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	ProgressTrackerDefaultReportFrequency = 100_000 // in blocks
)

func newProgressTracker[T any](cfg *utils.Config, reportFrequency int, log logger.Logger) *progressTracker[T] {
	return &progressTracker[T]{
		cfg:             cfg,
		log:             log,
		reportFrequency: reportFrequency,
	}
}

type progressTracker[T any] struct {
	extension.NilExtension[T]
	cfg                 *utils.Config
	log                 logger.Logger
	reportFrequency     int
	startOfRun          time.Time
	startOfLastInterval time.Time

	lock sync.Mutex
}

func (t *progressTracker[T]) PreRun(executor.State[T], *executor.Context) error {
	now := time.Now()
	t.startOfRun = now
	t.startOfLastInterval = now
	return nil
}
