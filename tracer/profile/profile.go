package profile

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

// ProfileStats data structure contains statedb operation statistics.
type ProfileStats struct {
	opFrequency   map[byte]uint64        // operation frequency stats
	opDuration    map[byte]time.Duration // accumulated operation duration
	opMinDuration map[byte]time.Duration // min runtime observerd
	opMaxDuration map[byte]time.Duration // max runtime observerd
	opVariance    map[byte]float64       // duration variance
	opLabel       map[byte]string        // operation names
	opOrder       []byte                 // order of map keys
	csv           string                 // csv file containing profiling data
	writeToFile   bool                   // if true print profiling results to a file
	hasHeader     bool                   // if write to a file, header prints once
}

func NewProfileStats(filename string) *ProfileStats {
	ps := new(ProfileStats)
	ps.Reset()
	ps.opOrder = make([]byte, 1)
	ps.csv = filename
	if filename != "" {
		ps.writeToFile = true
	}
	return ps
}

// Reset clears content in stats arrays
func (ps *ProfileStats) Reset() {
	ps.opFrequency = make(map[byte]uint64)
	ps.opDuration = make(map[byte]time.Duration)
	ps.opMinDuration = make(map[byte]time.Duration)
	ps.opMaxDuration = make(map[byte]time.Duration)
	ps.opVariance = make(map[byte]float64)
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

func (ps *ProfileStats) FillLabels(labels map[byte]string) {
	ps.opLabel = labels
	ps.opOrder = make([]byte, 0, len(ps.opLabel))
	for k := range ps.opLabel {
		ps.opOrder = append(ps.opOrder, k)
	}
	sort.Slice(ps.opOrder, func(i int, j int) bool { return ps.opOrder[i] < ps.opOrder[j] })
}

// PrintProfiling prints profiling information for executed operation.
func (ps *ProfileStats) PrintProfiling(first uint64, last uint64) error {
	var (
		builder strings.Builder
	)
	timeUnit := float64(time.Microsecond)
	if !ps.hasHeader {
		builder.WriteString("id, first, last, n, mean(us), std(us), min(us), max(us)\n")
		if ps.writeToFile {
			ps.hasHeader = true
		}
	}
	total := float64(0)
	for _, id := range ps.opOrder {
		if n, found := ps.opFrequency[id]; found {
			label := ps.opLabel[id]
			mean := (float64(ps.opDuration[id]) / float64(n)) / timeUnit
			std := math.Sqrt(ps.opVariance[id]) / timeUnit
			min := float64(ps.opMinDuration[id]) / timeUnit
			max := float64(ps.opMaxDuration[id]) / timeUnit
			fmt.Fprintf(&builder, "%v,%v,%v,%v,%v,%v,%v,%v\n", label, first, last, n, mean, std, min, max)

			total += float64(ps.opDuration[id])
		}
	}
	if ps.writeToFile {
		return ps.writeCsv(builder)
	} else {
		sec := total / float64(time.Second)
		fmt.Fprintf(&builder, "Total StateDB net execution time=%v (s)\n", sec)
		fmt.Println(builder.String())
	}
	return nil
}

// writeCsv writes stats to a file
func (ps *ProfileStats) writeCsv(builder strings.Builder) error {
	file, err := os.OpenFile(ps.csv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to print profiling; %v", err)
	}
	defer file.Close()
	file.WriteString(builder.String())
	return nil
}
