package analytics

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

type IncrementalStats struct {
	count uint64
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

func (s *IncrementalStats) ifEmpty(empty, notEmpty float64) float64 {
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

	// kahan sum
	y := x - s.c
	z := s.ksum + y
	s.c = (z - s.ksum) - y
	s.ksum = z

	s.min = s.ifEmpty(x, min(s.min, x))
	s.max = max(s.max, x)
	s.count += 1

}

func (s *IncrementalStats) GetCount() uint64 {
	return s.count
}

func (s *IncrementalStats) GetSum() float64 {
	return s.ksum
}

func (s *IncrementalStats) GetMean() float64 {
	return s.m1
}

func (s *IncrementalStats) GetVariance() float64 {
	return s.m2 / (float64(s.count))
}

func (s *IncrementalStats) GetStandardDeviation() float64 {
	return math.Sqrt(s.GetVariance())
}

func (s *IncrementalStats) GetSkewness() float64 {
	return math.Sqrt(float64(s.count)) * s.m3 / math.Pow(s.m2, 1.5)
}

func (s *IncrementalStats) GetKurtosis() float64 {
	return float64(s.count)*s.m4/(s.m2*s.m2) - 3.0
}

func (s *IncrementalStats) GetMin() float64 {
	return s.ifEmpty(math.NaN(), s.min)
}

func (s *IncrementalStats) GetMax() float64 {
	return s.ifEmpty(math.NaN(), s.max)
}

func (s *IncrementalStats) String() string {
	str, _ := json.Marshal(s)
	return string(str)
}

type IncrementalAnalytics struct {
	stats []IncrementalStats
}

func NewIncrementalAnalytics(opCount int) *IncrementalAnalytics {
	a := &IncrementalAnalytics{}
	a.stats = make([]IncrementalStats, opCount)
	return a
}

func (a *IncrementalAnalytics) Iterate() []IncrementalStats {
	return a.stats
}

func (a *IncrementalAnalytics) Reset() {
	a.stats = make([]IncrementalStats, len(a.stats))
}

func (a *IncrementalAnalytics) Update(id byte, data float64) {
	a.stats[id].Update(data)
}

func (a *IncrementalAnalytics) GetCount(id byte) uint64 {
	return a.stats[id].GetCount()
}

func (a *IncrementalAnalytics) GetMin(id byte) float64 {
	return a.stats[id].GetMin()
}

func (a *IncrementalAnalytics) GetMax(id byte) float64 {
	return a.stats[id].GetMax()
}

func (a *IncrementalAnalytics) GetSum(id byte) float64 {
	return a.stats[id].GetSum()
}

func (a *IncrementalAnalytics) GetMean(id byte) float64 {
	return a.stats[id].GetMean()
}

func (a *IncrementalAnalytics) GetVariance(id byte) float64 {
	return a.stats[id].GetVariance()
}

func (a *IncrementalAnalytics) GetStandardDeviation(id byte) float64 {
	return a.stats[id].GetStandardDeviation()
}

func (a *IncrementalAnalytics) GetSkewness(id byte) float64 {
	return a.stats[id].GetSkewness()
}

func (a *IncrementalAnalytics) GetKurtosis(id byte) float64 {
	return a.stats[id].GetKurtosis()
}
