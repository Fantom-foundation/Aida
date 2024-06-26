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

package analytics

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"gonum.org/v1/gonum/stat"
)

//
// General Helper Functions
//

// Assert if two objects are exactly the same
func assertExactlyEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("%s != %s", a, b)
	}
}

// Assert that the difference between two floats are negligible, e.g. within a certain threshold
const float64AlmostEqualThreshold = 1e-3

func assertAlmostEqual(t *testing.T, a float64, b float64) {
	if a == 0 {
		if b > float64AlmostEqualThreshold {
			t.Errorf("%f !~ %f", a, b)
		}
	} else {
		if math.Abs(a-b)/a > float64AlmostEqualThreshold {
			t.Log(a, b, math.Abs(a-b)/a, math.Abs(a-b)/a > float64AlmostEqualThreshold)
			t.Errorf("%f !~ %f", a, b)
		}
	}
}

// Assert if a float64 is NaN
func assertIsNaN(t *testing.T, a float64) {
	if !math.IsNaN(a) {
		t.Errorf("%.6f should have been NaN but isn't", a)
	}
}

// Assert if Kahan Sum is more corect than naive sum
func assertKahanSumMoreCorrect(t *testing.T, actual float64, sum float64, ksum float64) float64 {
	diff := math.Abs(actual - sum)
	kdiff := math.Abs(actual - ksum)

	if kdiff > diff {
		t.Errorf("kahan sum %.15f is further from %f (%.15f) than %.15f (%.15f)", ksum, actual, kdiff, sum, diff)
	}

	return ksum - sum
}

//
// Testcases
//

// Generate a large amount of a constant value to see if golang suffers from mathematical instability
func TestAnalyticsWithConstants(t *testing.T) {
	type result struct {
		mean     float64
		variance float64
		min      float64
		max      float64
	}

	type argument struct {
		count    uint64
		constant float64
	}

	type testcase struct {
		args argument
		want result
	}

	tests := []testcase{
		{args: argument{1e7, 1}, want: result{1, 0, 1, 1}},
		{args: argument{1e7, 1e-7}, want: result{1e-7, 0, 1e-7, 1e-7}},
		{args: argument{1e7, 1e7}, want: result{1e7, 0, 1e7, 1e7}},
	}

	type Sut struct {
		name string
		a    *IncrementalAnalytics
	}

	suts := []Sut{
		{name: "IncrementalAnalytics", a: NewIncrementalAnalytics(1)},
	}

	for _, test := range tests {
		for _, sut := range suts {
			name := fmt.Sprintf("[%s] WithConstant [%v]", sut.name, test.args)
			t.Run(name, func(t *testing.T) {
				a := sut.a
				a.Reset()

				for i := uint64(0); i < test.args.count; i++ {
					a.Update(0, test.args.constant)
				}

				got := result{
					a.GetMean(0),
					a.GetVariance(0),
					a.GetMin(0),
					a.GetMax(0),
				}

				assertExactlyEqual(t, test.want, got)
				assertExactlyEqual(t, test.args.count, a.GetCount(0))
				assertIsNaN(t, a.GetSkewness(0)) // Constant means there is undefined skewness
				assertIsNaN(t, a.GetKurtosis(0)) // Constant means there is undefined kurtosis
			})
		}
	}
}

// Generate a cycle of two values, to create scenario where mathematical instability would be more likely
func TestAnalyticsWithAlternativeBigSmall(t *testing.T) {
	type result struct {
		mean     float64
		variance float64
	}

	type argument struct {
		cycleCount    int
		big           float64
		bigPerCycle   int
		small         float64
		smallPerCycle int
	}

	type testcase struct {
		args argument
		want result
	}

	tests := []testcase{
		// one big value follows by large amount of very small values
		{args: argument{1, 1e10, 1, 1e-10, 1e6 - 1}, want: result{10000, 1e14 - 1e7}},

		// multiple cycles of two values
		{args: argument{1e4, 1e6, 1e3, -1e6, 1e3}, want: result{0, 1e12}},
	}

	type Sut struct {
		name string
		a    Analytics
	}

	suts := []Sut{
		{name: "IncrementalAnalytics", a: NewIncrementalAnalytics(1)},
	}

	for _, test := range tests {
		for _, sut := range suts {
			name := fmt.Sprintf("[%s] WithAlternativeBigSmall [%v]", sut.name, test.args)
			t.Run(name, func(t *testing.T) {
				a := sut.a
				a.Reset()

				for i := 0; i < test.args.cycleCount; i++ {
					for j := 0; j < test.args.bigPerCycle; j++ {
						a.Update(0, test.args.big)
					}
					for j := 0; j < test.args.smallPerCycle; j++ {
						a.Update(0, test.args.small)
					}
				}
				got := result{a.GetMean(0), a.GetVariance(0)}

				n := uint64(test.args.cycleCount * (test.args.bigPerCycle + test.args.smallPerCycle))
				assertExactlyEqual(t, n, a.GetCount(0))
				assertAlmostEqual(t, test.want.mean, got.mean)
				assertAlmostEqual(t, test.want.variance, got.variance)
			})
		}
	}
}

// Generate using a defined gaussian distributions, and using the observations to reason about the population
func TestAnalyticsWithGaussianDistribution(t *testing.T) {
	type argument struct {
		seed     int64 // -1 = don't use my seed, random something for me
		amount   uint64
		mean     float64
		variance float64
	}

	type result struct {
		mean     float64
		variance float64
	}

	type testcase struct {
		args argument
		want result
	}

	tests := []testcase{
		{args: argument{258, 1e8, 0, 1}, want: result{0, 1}},
		{args: argument{258, 1e8, 25, 10000}, want: result{25, 10000}},
		{args: argument{258, 1e8, 1e10, 1e20}, want: result{1e10, 1e20}},
	}

	type Sut struct {
		name string
		a    Analytics
	}

	suts := []Sut{
		{name: "IncrementalAnalytics", a: NewIncrementalAnalytics(1)},
	}

	for _, test := range tests {
		for _, sut := range suts {
			name := fmt.Sprintf("[%s] WithGaussian [%+v]", sut.name, test.args)
			t.Run(name, func(t *testing.T) {
				a := sut.a
				a.Reset()

				var r *rand.Rand
				if test.args.seed != 0 && test.args.seed != -1 {
					r = rand.New(rand.NewSource(test.args.seed))
				} else {
					r = rand.New(rand.NewSource(time.Now().UnixNano()))
				}

				for i := uint64(0); i < test.args.amount; i++ {
					x := r.NormFloat64()*math.Sqrt(test.args.variance) + test.args.mean
					a.Update(0, x)
				}
				got := result{a.GetMean(0), a.GetVariance(0)}

				assertAlmostEqual(t, test.want.mean, got.mean)
				assertAlmostEqual(t, test.want.variance, got.variance)
				assertAlmostEqual(t, 0, a.GetSkewness(0)) // skewness should be close to 0
				assertAlmostEqual(t, 0, a.GetKurtosis(0)) // kurtosis should be close to 0
				assertExactlyEqual(t, test.args.amount, a.GetCount(0))
			})
		}
	}
}

// Helper function to calculate Kurtosis manually
// Majority of the libraries calculate "sample kurtosis", but our algorithm calculates "population kurtosis"
// In order to check correctness of the kurtosis calculation, it's manually calculated here
// Sample Kurtosis is compared to external library and Population Kurtosis is compared against the algorithm
func calculateKurtosis(data []float64) (float64, float64) {
	n := float64(len(data))
	mean := stat.Mean(data, nil)
	variance := stat.Variance(data, nil)
	s_std := math.Sqrt(variance)
	p_std := math.Sqrt(variance * (n - 1) / (n))

	var k, l float64
	for _, x := range data {
		k += math.Pow(((x - mean) / s_std), 4)
		l += math.Pow(((x - mean) / p_std), 4)
	}

	sk := k*n*(n+1)/(n-1)/(n-2)/(n-3) - 3*(n-1)*(n-1)/(n-2)/(n-3)
	pk := l/n - 3

	return sk, pk
}

// Generate small sequence with known results to measure correctness of calculation of mean, variance, std
func TestAnalyticsWithKnownInput(t *testing.T) {
	type result struct {
		mean     float64
		variance float64
		skewness float64
		kurtosis float64
	}

	type testcase struct {
		args []float64
	}

	tests := []testcase{
		{args: []float64{10, 20, 30, 20}},
		{args: []float64{10, 20, 30, 20, 10, 20, 30, 20}},
		{args: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}},
		{args: []float64{10, 20, 30, 30, 30, 30, 30, 30}},
		{args: []float64{1.1, 3.345, 12.234, 11.945, 14.235, 16.876, 20.213, 11.001, 7.098, 21.234}},
	}

	type Sut struct {
		name string
		a    Analytics
	}

	suts := []Sut{
		{name: "IncrementalAnalytics", a: NewIncrementalAnalytics(1)},
	}

	for _, test := range tests {
		for _, sut := range suts {
			name := fmt.Sprintf("[%s] WithKnownInput [%+v]", sut.name, test.args)
			t.Run(name, func(t *testing.T) {

				a := sut.a
				a.Reset()

				for _, x := range test.args {
					a.Update(0, x)
				}

				n := float64(len(test.args))
				want := result{
					stat.Mean(test.args, nil),
					stat.Variance(test.args, nil) * (n - 1) / n,
					stat.Skew(test.args, nil) * (n - 2) / math.Sqrt(n*(n-1)),
					stat.ExKurtosis(test.args, nil),
				}

				got := result{
					a.GetMean(0),
					a.GetVariance(0),
					a.GetSkewness(0),
					a.GetKurtosis(0),
				}

				sk, pk := calculateKurtosis(test.args)

				assertExactlyEqual(t, uint64(len(test.args)), a.GetCount(0))
				assertAlmostEqual(t, want.mean, got.mean)
				assertAlmostEqual(t, want.variance, got.variance)
				assertAlmostEqual(t, want.skewness, got.skewness)
				assertAlmostEqual(t, want.kurtosis, sk)
				assertAlmostEqual(t, got.kurtosis, pk)
			})
		}
	}

}
