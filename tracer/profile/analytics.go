package profile

import (
	"math"
	"encoding/json"
)

func min (a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max (a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

type Analytics []IncrementalStats

/*
func NewAnalytics(opCount uint64) *Analytics {
	return &Analytics{make([]IncrementalStats, opCount)}
}
*/

type IncrementalStats struct {
	count		uint64
	sum		float64
	variance	float64
	min		float64
	max		float64
}

func NewIncrementalStats() *IncrementalStats {
	return &IncrementalStats{variance: math.NaN()}
}

func (s *IncrementalStats) unlessEmpty(notEmpty, empty float64) float64 {
	if s.count != 0 {
		return notEmpty
	}
	return empty
}

func (s *IncrementalStats) Update(x float64) {
	oldCount, newCount := float64(s.count), float64(s.count + 1)
	oldSum, newSum := s.sum, s.sum + x

	oldMean := s.unlessEmpty(oldSum / oldCount, 0)
	//newMean := newSum / newCount
	
	diff := oldMean - x
	oldVariance := s.variance
	newVariance := s.unlessEmpty(
		//oldVariance * (oldCount - 1) / oldCount + diffMean * diffMean / newCount,
		(oldCount / newCount) * (oldVariance + (diff * diff / newCount)),
		0,
	)

	s.count = uint64(newCount)
	s.sum = newSum
	s.variance = newVariance
	//a.min = oldCount > 0 ? min(a.min, x) : x
	s.min = s.unlessEmpty( min(s.min, x), x)
	s.max = max(s.max, x)
}

func (s *IncrementalStats) GetCount() uint64 {
	return s.count
}

func (s *IncrementalStats) GetSum() float64 {
	return s.sum
}

func (s *IncrementalStats) GetMean() float64 {
	return s.unlessEmpty( s.sum/float64(s.count), 0)
}

func (s *IncrementalStats) GetVariance() float64 {
	return s.variance	
}

func (s *IncrementalStats) GetMin() float64 {
	return s.min
}

func (s *IncrementalStats) GetMax() float64 {
	return s.max	
}

func (s *IncrementalStats) String() string {
	str, _ := json.Marshal(s)
	return string(str)
}
