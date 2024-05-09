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

package utils

import (
	"time"

	"github.com/Fantom-foundation/Aida/logger"
)

// threshold for wrapping a bulk load and reporting a priming progress
const OperationThreshold = 1_000_000

type ProgressTracker struct {
	step   int           // step counter
	target int           // total number of steps
	start  time.Time     // start time
	last   time.Time     // last reported time
	rate   float64       // priming rate
	log    logger.Logger // Message logger
}

// NewProgressTracker creates a new progress tracer
func NewProgressTracker(target int, log logger.Logger) *ProgressTracker {
	now := time.Now()
	return &ProgressTracker{
		step:   0,
		target: target,
		start:  now,
		last:   now,
		rate:   0.0,
		log:    log,
	}
}

// PrintProgress reports a priming rates and estimated time after n operations has been executed.
func (pt *ProgressTracker) PrintProgress() {
	pt.step++
	if pt.step%OperationThreshold == 0 {
		now := time.Now()
		currentRate := OperationThreshold / now.Sub(pt.last).Seconds()
		pt.rate = currentRate*0.1 + pt.rate*0.9
		pt.last = now
		progress := float32(pt.step) / float32(pt.target)
		time := int(now.Sub(pt.start).Seconds())
		eta := int(float64(pt.target-pt.step) / pt.rate)
		pt.log.Infof("\t\tLoading state ... %8.1f slots/s, %5.1f%%, time: %d:%02d, ETA: %d:%02d", currentRate, progress*100, time/60, time%60, eta/60, eta%60)
	}
}
