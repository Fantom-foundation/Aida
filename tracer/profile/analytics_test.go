package profile

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/stat"
)

func assertExactlyEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("%s != %s", a, b)
	}
}

const float64AlmostEqualThreshold = 1e-3

func assertAlmostEqual(t *testing.T, a float64, b float64) {
	if a == 0 {
		if b > float64AlmostEqualThreshold {
			t.Errorf("%f !~ %f", a, b)
		}
	} else {
		if math.Abs(a - b) / a > float64AlmostEqualThreshold {
			t.Log(a, b, math.Abs(a-b) / a, math.Abs(a-b) / a > float64AlmostEqualThreshold)
			t.Errorf("%f !~ %f", a, b)
		}
	}

}

func assertIsNaN(t *testing.T, a float64) {
	if !math.IsNaN(a) {
		t.Errorf("%.6f should have been NaN but isn't", a)
	}
}

func TestAnalyticsWithConstants(t *testing.T) {
	type result struct {
		mean     float64
		variance float64
	}

	type argument struct {
		count		uint64
		constant	float64
	}

	type testcase struct {
		args argument
		want result
	}

	tests := []testcase{
		{args: argument{1e7, 1}, want: result{1, 0}},
		{args: argument{1e7, 1e-7}, want: result{1e-7, 0}},
		{args: argument{1e7, 1e7}, want: result{1e7, 0}},
	}

	for _, test := range tests {
		name := fmt.Sprintf("AnalyticsWithConstant [%v]", test.args)
		t.Run(name, func(t *testing.T) {
			a := [1]IncrementalStats{}
			for i := uint64(0); i < test.args.count; i++ {
				a[0].Update(test.args.constant)
			}
			got := result{
				a[0].GetMean(),
				a[0].GetVariance(),
			}

			assertExactlyEqual(t, test.want, got)
			assertExactlyEqual(t, test.args.count, a[0].GetCount())
			assertIsNaN(t, a[0].GetSkewness())
			assertIsNaN(t, a[0].GetKurtosis())
			assertExactlyEqual(t, a[0].GetSum(), a[0].getKahanSum())
		})
	}
}

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
		{args: argument{1, 1e10, 1, 1e-10, 1e6 - 1}, want: result{10000, 1e14-1e7}},
		{args: argument{1e4, 1e6, 1e3, -1e6, 1e3}, want: result{0, 1e12}},
	}

	for _, test := range tests {
		name := fmt.Sprintf("AnalyticsWithAlternativeBigSmall [%v]", test.args)
		t.Run(name, func(t *testing.T) {
			a := [1]IncrementalStats{}
			for i := 0; i < test.args.cycleCount; i++ {
				for j := 0; j < test.args.bigPerCycle; j++ {
					a[0].Update(test.args.big)
				}
				for j := 0; j < test.args.smallPerCycle; j++ {
					a[0].Update(test.args.small)
				}
			}
			got := result{a[0].GetMean(), a[0].GetVariance()}

			assertExactlyEqual(t, uint64(test.args.cycleCount * (test.args.bigPerCycle + test.args.smallPerCycle)), a[0].GetCount())
			assertAlmostEqual(t, test.want.mean, got.mean)
			assertAlmostEqual(t, test.want.variance, got.variance)
			assertExactlyEqual(t, a[0].GetSum(), a[0].getKahanSum())
		})
	}
}

func TestAnalyticsWithGaussianDistribution(t *testing.T) {
	type argument struct {
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
		{args: argument{1e8, 0, 1}, want: result{0, 1}},
		{args: argument{1e8, 25, 10000}, want: result{25, 10000}},
	}

	for _, test := range tests {
		name := fmt.Sprintf("AnalyticsWithGaussian [%+v]", test.args)
		t.Run(name, func(t *testing.T) {
			a := [1]IncrementalStats{}
			for i := uint64(0); i < test.args.amount; i++ {
				x := rand.NormFloat64()*math.Sqrt(test.args.variance) + test.args.mean
				a[0].Update(x)
			}
			got := result{a[0].GetMean(), a[0].GetVariance()}

			//assertAlmostEqual(t, test.want, got)
			assertAlmostEqual(t, test.want.mean, got.mean)
			assertAlmostEqual(t, test.want.variance, got.variance)
			assertExactlyEqual(t, test.args.amount, a[0].GetCount())
			assertExactlyEqual(t, a[0].GetSum(), a[0].getKahanSum())
		})
	}
}

func calculateKurtosis(data []float64) (float64, float64) {
	n := float64(len(data))
	mean := stat.Mean(data, nil)
	variance := stat.Variance(data, nil)
	s_std := math.Sqrt(variance)
	p_std := math.Sqrt(variance * (n-1) / (n))
	
	var k, l float64
	for _, x := range data {
		k += math.Pow( ((x - mean) / s_std), 4)
		l += math.Pow( ((x - mean) / p_std), 4)
	}

	sk := k * n * (n+1) / (n-1) / (n-2) / (n-3) - 3 * (n-1) * (n-1) / (n-2) /  (n-3)
	pk := l / n - 3

	return sk, pk
}

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

	for _, test := range tests {
		name := fmt.Sprintf("AnalyticsWithKnownInput [%+v]", test.args)
		t.Run(name, func(t *testing.T) {
			
			a := [1]IncrementalStats{}
			for _, x := range test.args {
				a[0].Update(x)
			}

			n := float64(len(test.args))
			want := result {
				stat.Mean(test.args, nil),
				stat.Variance(test.args, nil) * (n-1) / n ,
				stat.Skew(test.args, nil) * (n-2) / math.Sqrt(n*(n-1)), 
				stat.ExKurtosis(test.args, nil),
			}

			got := result{
				a[0].GetMean(),
				a[0].GetVariance(),
				a[0].GetSkewness(),
				a[0].GetKurtosis(),
			}

			sk, pk := calculateKurtosis(test.args)

			assertExactlyEqual(t, uint64(len(test.args)), a[0].GetCount())
			assertAlmostEqual(t, want.mean, got.mean)
			assertAlmostEqual(t, want.variance, got.variance)
			assertAlmostEqual(t, want.skewness, got.skewness)
			assertAlmostEqual(t, want.kurtosis, sk)
			assertAlmostEqual(t, got.kurtosis, pk)
			assertExactlyEqual(t, a[0].GetSum(), a[0].getKahanSum())
		})
	}

}
