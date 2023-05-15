package operation

import (
	"math"
	"time"

	"github.com/op/go-logging"
)

var EnableProfiling = false

// ProfileStats data structure contains statedb operation statistics.
type ProfileStats struct {
	opFrequency   [NumOperations]uint64        // operation frequency stats
	opDuration    [NumOperations]time.Duration // accumulated operation duration
	opMinDuration [NumOperations]time.Duration // min runtime observerd
	opMaxDuration [NumOperations]time.Duration // max runtime observerd
	opVariance    [NumOperations]float64       // duration variance
}

// Profiling records runtime and calculates statistics after
// executing a state operation.
func (ps *ProfileStats) Profile(id byte, elapsed time.Duration) {
	n := ps.opFrequency[id]
	duration := ps.opDuration[id]
	// update min/max values
	if n > 0 {
		if ps.opMaxDuration[id] < elapsed {
			ps.opMaxDuration[id] = elapsed
		}
		if ps.opMinDuration[id] > elapsed {
			ps.opMinDuration[id] = elapsed
		}
	} else {
		ps.opMinDuration[id] = elapsed
		ps.opMaxDuration[id] = elapsed
	}
	// compute previous mean
	prevMean := float64(0.0)
	if n > 0 {
		prevMean = float64(ps.opDuration[id]) / float64(n)
	}
	// update variance
	newDuration := duration + elapsed
	if n > 0 {
		newMean := float64(newDuration) / float64(n+1)
		ps.opVariance[id] = float64(n-1)*ps.opVariance[id]/float64(n) +
			(newMean-prevMean)*(newMean-prevMean)/float64(n+1)
	} else {
		ps.opVariance[id] = 0.0
	}

	// update execution frequency
	ps.opFrequency[id] = n + 1

	// update accumulated duration and frequency
	ps.opDuration[id] = newDuration
}

// PrintProfiling prints profiling information for executed operation.
func (ps *ProfileStats) PrintProfiling(log *logging.Logger) {
	timeUnit := float64(time.Microsecond)
	tuStr := "us"
	log.Noticef("id, n, mean(%v), std(%v), min(%v), max(%v)\n", tuStr, tuStr, tuStr, tuStr)
	total := float64(0)
	for id := byte(0); id < NumOperations; id++ {
		n := ps.opFrequency[id]
		mean := (float64(ps.opDuration[id]) / float64(n)) / timeUnit
		std := math.Sqrt(ps.opVariance[id]) / timeUnit
		min := float64(ps.opMinDuration[id]) / timeUnit
		max := float64(ps.opMaxDuration[id]) / timeUnit
		log.Noticef("%v, %v, %v, %v, %v, %v\n", GetLabel(id), n, mean, std, min, max)

		total += float64(ps.opDuration[id])
	}
	sec := total / float64(time.Second)
	tps := float64(ps.opFrequency[FinaliseID]) / sec
	log.Noticef("Total StateDB net execution time=%v (s) / ~%.1f Tx/s\n", sec, tps)
}
