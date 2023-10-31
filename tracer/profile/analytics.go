package profile

import (
	"encoding/json"
	"math"
)

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

type Analytics []IncrementalStats

/*
func NewAnalytics(opCount uint64) *Analytics {
	return &Analytics{ make([]IncrementalStats, opCount){} }
}
*/

type IncrementalStats struct {
	count uint64
	sum   float64
	min   float64
	max   float64

	ksum float64
	c    float64

	m1 float64
	m2 float64
	m3 float64
	m4 float64
}

func NewIncrementalStats() *IncrementalStats {
	return &IncrementalStats{}
}

func (s *IncrementalStats) unlessEmpty(notEmpty, empty float64) float64 {
	if s.count != 0 {
		return notEmpty
	}
	return empty
}

func (s *IncrementalStats) Update(x float64) {
	prevN, n := float64(s.count), float64(s.count+1)

	delta := x - s.m1
	delta_n := delta / n
	delta_n2 := delta_n * delta_n

	t := delta * delta_n * prevN
	s.m1 += delta_n
	s.m4 += t*delta_n2*(n*n-3*n+3) + (6 * delta_n2 * s.m2) - (4 * delta_n * s.m3)
	s.m3 += t*delta_n*(n-2) - (3 * delta_n * s.m2)
	s.m2 += t

	s.count += 1
	s.sum += x
	s.min = s.unlessEmpty(min(s.min, x), x)
	s.max = max(s.max, x)

	//kahan sum
	y := x - s.c
	z := s.ksum + y
	s.c = (z - s.ksum) - y
	s.ksum = z
}

func (s *IncrementalStats) GetCount() uint64 {
	return s.count
}

func (s *IncrementalStats) GetSum() float64 {
	return s.sum
}

func (s *IncrementalStats) getKahanSum() float64 {
	return s.ksum
}

func (s *IncrementalStats) GetMean() float64 {
	return s.m1
}

func (s *IncrementalStats) GetVariance() float64 {
	return s.m2 / (float64(s.count))
}

func (s *IncrementalStats) GetSkewness() float64 {
	return math.Sqrt(float64(s.count)) * s.m3 / math.Pow(s.m2, 1.5)
}

func (s *IncrementalStats) GetKurtosis() float64 {
	return float64(s.count)*s.m4/(s.m2*s.m2) - 3.0
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
