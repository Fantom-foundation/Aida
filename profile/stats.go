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

package profile

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

// Stats data structure contains statedb operation statistics.
type Stats struct {
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

func NewStats(filename string) *Stats {
	ps := new(Stats)
	ps.Reset()
	ps.opOrder = make([]byte, 1)
	ps.csv = filename
	if filename != "" {
		ps.writeToFile = true
	}
	return ps
}

// Reset clears content in stats arrays
func (ps *Stats) Reset() {
	ps.opFrequency = make(map[byte]uint64)
	ps.opDuration = make(map[byte]time.Duration)
	ps.opMinDuration = make(map[byte]time.Duration)
	ps.opMaxDuration = make(map[byte]time.Duration)
	ps.opVariance = make(map[byte]float64)
}

// Profiling records runtime and calculates statistics after
// executing a state operation.
func (ps *Stats) Profile(id byte, elapsed time.Duration) {
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

func (ps *Stats) FillLabels(labels map[byte]string) {
	ps.opLabel = labels
	ps.opOrder = make([]byte, 0, len(ps.opLabel))
	for k := range ps.opLabel {
		ps.opOrder = append(ps.opOrder, k)
	}
	sort.Slice(ps.opOrder, func(i int, j int) bool { return ps.opOrder[i] < ps.opOrder[j] })
}

// PrintProfiling prints profiling information for executed operation.
func (ps *Stats) PrintProfiling(first uint64, last uint64) error {
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
func (ps *Stats) writeCsv(builder strings.Builder) error {
	file, err := os.OpenFile(ps.csv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to print profiling; %v", err)
	}
	defer file.Close()
	file.WriteString(builder.String())
	return nil
}

// Added these so we can test the stats
func (ps *Stats) GetOpOrder() []byte {
	return ps.opOrder
}

type Stat struct {
	Frequency   uint64        // operation frequency stats
	Duration    time.Duration // accumulated operation duration
	MinDuration time.Duration // min runtime observerd
	MaxDuration time.Duration // max runtime observerd
	Variance    float64       // duration variance
	Label       string        // operation names

}

func (ps *Stats) GetStatByOpId(op byte) *Stat {
	return &Stat{
		Frequency:   ps.opFrequency[op],
		Duration:    ps.opDuration[op],
		MinDuration: ps.opMinDuration[op],
		MaxDuration: ps.opMaxDuration[op],
		Variance:    ps.opVariance[op],
		Label:       ps.opLabel[op],
	}
}

func (ps *Stats) GetTotalOpFreq() int {
	totalOpFreq := int(0)
	for _, freq := range ps.opFrequency {
		totalOpFreq += int(freq)
	}
	return totalOpFreq
}
