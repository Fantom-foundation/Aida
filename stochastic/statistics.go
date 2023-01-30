package stochastic

import (
	"sort"
)

// numPoints determines approximate number of the points for the
// empirical cumulative distribution.
const numPoints = 100

// Statistics for counting frequencies of data.
type Statistics[T comparable] struct {
	distribution map[T]uint64 // frequencies of hashes
}

// StatisticsDistribution for JSON
type StatisticsDistribution struct {
	NumData   int
	TotalFreq uint64
	ECdf      [][2]float64 // Value of random variable
}

// NewStatistics creates a new statistics.
func NewStatistics[T comparable]() Statistics[T] {
	return Statistics[T]{map[T]uint64{}}
}

// Count increments the distribution of a hash by one.
func (s *Statistics[T]) Count(data T) {
	s.distribution[data]++
}

// Frequency returns the count distribution of a hash value.
func (s *Statistics[T]) Frequency(data T) uint64 {
	return s.distribution[data]
}

// Exists check whether hash exists in the statistics.
func (s *Statistics[T]) Exists(data T) bool {
	_, ok := s.distribution[data]
	return ok
}

// Write empirical cumulative distribution as as comma-separated file.
// We write out only 100 equi-distant data points
func (s *Statistics[T]) ProduceDistribution() StatisticsDistribution {

	// sort data according to their ascending frequency
	// and compute totalFreq frequency.
	numData := len(s.distribution)
	entries := make([]T, 0, numData)
	totalFreq := uint64(0)
	for data, freq := range s.distribution {
		entries = append(entries, data)
		totalFreq += freq
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return s.distribution[entries[i]] > s.distribution[entries[j]]
	})

	ECdf := [][2]float64{}

	// if no data-points, nothing to plot
	if numPoints != 0 {

		// compute distance between points in the distribution
		// to limit the number of printed points
		d := numData / numPoints
		if d < 1 {
			d = 1
		}

		// print points of the empirical cumulative distribution
		// in subsequent rows.
		sum := float64(0.0)
		c := float64(0.0)
		ECdf = append(ECdf, [2]float64{0.0, 0.0})
		j := 0
		for i := 0; i < numData; i++ {
			// Implement Kahan's summation to avoid errors
			// for accumulated probabilities (they might be very small)
			// https://en.wikipedia.org/wiki/Kahan_summation_algorithm
			y := float64(s.distribution[entries[i]])/float64(totalFreq) - c
			t := sum + y
			c = (t - sum) - y
			sum = t
			if j < d {
				j++
			} else {
				ECdf = append(ECdf, [2]float64{(float64(i) + 0.5) / float64(numData), sum})
				j = 0
			}
		}
		ECdf = append(ECdf, [2]float64{1.0, 1.0})
	}

	return StatisticsDistribution{
		NumData:   numData,
		TotalFreq: totalFreq,
		ECdf:      ECdf,
	}
}
