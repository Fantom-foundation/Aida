package stochastic

import (
	"sort"
)

// CountingStats for counting frequencies of data items.
type CountingStats[T comparable] struct {
	freq map[T]uint64 // frequency counts per item
}

// JSON output for a CountingStats object
type CountingStatsJSON struct {
	NumKeys int          `json:"n"`    // Number of data entries
	ECdf    [][2]float64 `json:"ecdf"` // Empirical cumulative distribution function
}

// NewCountingStats creates a new counting statistics.
func NewCountingStats[T comparable]() CountingStats[T] {
	return CountingStats[T]{map[T]uint64{}}
}

// Count increments the frequency of a data item by one.
func (s *CountingStats[T]) Count(data T) {
	s.freq[data]++
}

// Frequency returns the count frequency of a data item value.
func (s *CountingStats[T]) Frequency(data T) uint64 {
	return s.freq[data]
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

	ECdf := [][2]float64{}

	// if no data-points, nothing to plot
	if numKeys > 0 {
		// plotting distance of points in eCDF
		d := numKeys / numPoints
		if d < 1 {
			d = 1
		}

		// print points of the empirical cumulative freq
		sumP := float64(0.0)

		// Correction term for Kahan's sum
		cP := float64(0.0)

		// Prime ECDF and counter
		ECdf = append(ECdf, [2]float64{0.0, 0.0})
		ctr := 1

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

			// Add only d-times a new point to the empirical cumulative
			// distribution function, i.e, points in the ECDF will be
			// equi-distant.
			if ctr < d {
				ctr++
			} else {
				ECdf = append(ECdf, [2]float64{x, sumP})
				ctr = 1
			}
		}
		// add last point
		ECdf = append(ECdf, [2]float64{1.0, 1.0})
	}

	return CountingStatsJSON{
		NumKeys: numKeys,
		ECdf:    ECdf,
	}
}

// NewCountingStatsJSON computes the ECDF of the counting stats.
func (s *CountingStats[T]) NewCountingStatsJSON() CountingStatsJSON {
	return s.produceJSON(numDistributionPoints)
}
