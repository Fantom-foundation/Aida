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
