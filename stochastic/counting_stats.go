package stochastic

import (
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/simplify"
)

// CountingStats for counting frequencies of data items.
type CountingStats[T comparable] struct {
	freq map[T]uint64 // frequency counts per item
}

// JSON output for a CountingStats object
type CountingStatsJSON struct {
	NumKeys int64        `json:"n"`    // Number of data entries
	ECdf    [][2]float64 `json:"ecdf"` // Empirical cumulative distribution function
}

// NewCountingStats creates a new counting statistics.
func NewCountingStats[T comparable]() CountingStats[T] {
	return CountingStats[T]{map[T]uint64{}}
}

// Places an item into the counting statistics.
func (s *CountingStats[T]) Place(data T) {
	s.freq[data]++
}

// Exists check whether data item exists in the counting statistics.
func (s *CountingStats[T]) Exists(data T) bool {
	_, ok := s.freq[data]
	return ok
}

// produceJSON computes the ECDF and set the number field in the JSON struct.
func (s *CountingStats[T]) produceJSON(numPoints int) CountingStatsJSON {

	// sort data according to their descending frequency
	// and compute totalFreq frequency.

	numKeys := len(s.freq)
	entries := make([]T, 0, numKeys)
	totalFreq := uint64(0)
	for data, freq := range s.freq {
		entries = append(entries, data)
		totalFreq += freq
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return s.freq[entries[i]] > s.freq[entries[j]]
	})

	// simplified eCDF
	var simplified orb.LineString

	// if no data-points, nothing to plot
	if numKeys > 0 {

		// construct full eCdf as LineString
		ls := orb.LineString{}

		// print points of the empirical cumulative freq
		sumP := float64(0.0)

		// Correction term for Kahan's sum
		cP := float64(0.0)

		// add first point to line string
		ls = append(ls, orb.Point{0.0, 0.0})

		// iterate through all items
		for i := 0; i < numKeys; i++ {
			// Implement Kahan's summation to avoid errors
			// for accumulated probabilities (they might be very small)
			// https://en.wikipedia.org/wiki/Kahan_summation_algorithm
			f := float64(s.freq[entries[i]]) / float64(totalFreq)
			x := (float64(i) + 0.5) / float64(numKeys)

			yP := f - cP
			tP := sumP + yP
			cP = (tP - sumP) - yP
			sumP = tP

			// add new point to Ecdf
			ls = append(ls, orb.Point{x, sumP})
		}

		// add last point
		ls = append(ls, orb.Point{1.0, 1.0})

		// reduce full ecdf using Visvalingam-Whyatt algorithm to
		// "numPoints" points. See:
		// https://en.wikipedia.org/wiki/Visvalingam-Whyatt_algorithm
		simplifier := simplify.VisvalingamKeep(numPoints)
		simplified = simplifier.Simplify(ls).(orb.LineString)
	}

	// convert orb.LineString to [][2]float64
	ECdf := make([][2]float64, len(simplified))
	for i := range simplified {
		ECdf[i] = [2]float64(simplified[i])
	}

	return CountingStatsJSON{
		NumKeys: int64(numKeys),
		ECdf:    ECdf,
	}
}

// NewCountingStatsJSON computes the ECDF of the counting stats.
func (s *CountingStats[T]) NewCountingStatsJSON() CountingStatsJSON {
	return s.produceJSON(numDistributionPoints)
}
