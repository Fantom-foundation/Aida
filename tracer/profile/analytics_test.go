package profile

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func assertExactlyEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("%s != %s", a, b)
	}
}

func assertAlmostEqual(t *testing.T, a float64, b float64) {
	roundedToThirdDecimal := func(f float64) string { return fmt.Sprintf("%.3f", f) }
	if roundedToThirdDecimal(a) != roundedToThirdDecimal(b) {
		t.Errorf("%.3f !~ %.3f", a, b)
	}
}

func assertIsNaN(t *testing.T, a float64) {
	if !math.IsNaN(a) {
		t.Errorf("%.6f should have been NaN but isn't", a)
	}
}

func TestAnalyticsWithOnes(t *testing.T) {
	type result struct {
		mean     float64
		variance float64
	}

	type testcase struct {
		args uint64
		want result
	}

	tests := []testcase{
		{args: 100, want: result{1, 0}},
		{args: 10_000, want: result{1, 0}},
		{args: 1_000_000, want: result{1, 0}},
		{args: math.MaxInt32, want: result{1, 0}},
	}

	for _, test := range tests {
		name := fmt.Sprintf("AnalyticsWithOnes [%d]", test.args)
		t.Run(name, func(t *testing.T) {
			a := [1]IncrementalStats{}
			for i := uint64(0); i < test.args; i++ {
				a[0].Update(1)
			}
			got := result{
				a[0].GetMean(),
				a[0].GetVariance(),
			}

			assertExactlyEqual(t, test.want, got)
			assertExactlyEqual(t, test.args, a[0].GetCount())
			assertIsNaN(t, a[0].GetSkewness())
			assertIsNaN(t, a[0].GetKurtosis())

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
		{args: argument{1, 1e10, 1, 1e-3, 1e10 - 1}, want: result{1.001, 0}},
		{args: argument{100, 1e10, 1, 1e-3, 1}, want: result{1, 0}},
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

			assertExactlyEqual(t, test.want, got)
			//assertExactlyEqual(t, test.args, a[0].GetCount())
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
		{args: argument{1_000_000, 10, 100}, want: result{10, 100}},
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
			assertExactlyEqual(t, test.args, a[0].GetCount())
		})
	}

}

func TestAnalyticsWithKnownInput(t *testing.T) {
	type argument []float64

	type result struct {
		mean     float64
		variance float64
		skewness float64
		kurtosus float64
	}

	type testcase struct {
		args argument
		want result
	}

	tests := []testcase{
		{args: []float64{10, 20, 30, 20, 10}, want: result{18, 56, 0.512241, -0.612245}},
		//{args: []float64{10, 20, 30, 20}, want: result{20, float64(200) / 4}},
		//{args: []float64{3, 8, 10, 17, 24, 27}, want: result{14.8333, 74.4722, 0.17487, -1.7511}},
	}

	for _, test := range tests {
		name := fmt.Sprintf("AnalyticsWithKnownInput [%+v]", test.args)
		t.Run(name, func(t *testing.T) {
			a := [1]IncrementalStats{}
			for _, x := range test.args {
				a[0].Update(x)
				t.Log(x)
				t.Log(a[0].GetCount(), a[0].GetMean(), a[0].GetVariance())
			}

			got := result{
				a[0].GetMean(),
				a[0].GetVariance(),
				a[0].GetSkewness(),
				a[0].GetKurtosis(),
			}

			assertExactlyEqual(t, uint64(len(test.args)), a[0].GetCount())
			assertExactlyEqual(t, test.want, got)
		})
	}

}
