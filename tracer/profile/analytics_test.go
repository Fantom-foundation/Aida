package profile

import (
	"testing"
	"fmt"
	"math"
	"math/rand"
)

func assertExactlyEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("%s != %s", a, b)
	}
}

func TestAnalyticsWithOnes(t *testing.T) {
	type result struct {
		mean float64
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
			for i := uint64(0) ; i < test.args ; i++ {
				a[0].Update(1)
			}
			got := result{a[0].GetMean(), a[0].GetVariance()}
			
			assertExactlyEqual(t, test.want, got)
			assertExactlyEqual(t, test.args, a[0].GetCount())
		})
	}
}

func TestAnalyticsWithGaussianDistrbution(t *testing.T) {
	type argument struct {
		amount uint64
		mean float64
		variance float64
	}

	type result struct {
		mean float64
		variance float64
	}

	type testcase struct {
		args argument
		want result
	}

	tests := []testcase {
		{args: argument{1_000_000, 10, 100}, want: result{10, 100}},
	}

	for _, test := range tests {
		name := fmt.Sprintf("AnalyticsWithGaussian [%+v]", test.args)
		t.Run(name, func(t *testing.T) {
			a := [1]IncrementalStats{}
			for i := uint64(0) ; i < test.args.amount ; i++ {
				x := rand.NormFloat64() * test.args.variance + test.args.mean
				a[0].Update(x)
			}
			got := result{a[0].GetMean(), a[0].GetVariance()}
			
			assertExactlyEqual(t, test.want, got)
			assertExactlyEqual(t, test.args, a[0].GetCount())
		})
	}

}

func TestAnalyticsWithKnownInput(t *testing.T) {
	type argument []float64

	type result struct {
		mean float64
		variance float64
	}

	type testcase struct {
		args argument
		want result
	}

	tests := []testcase {
		{args: []float64{10, 20, 30}, want: result{20, float64(200)/3}},
		{args: []float64{10, 20, 30, 20}, want: result{20, float64(200)/4}},
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
			got := result{a[0].GetMean(), a[0].GetVariance()}
			
			assertExactlyEqual(t, uint64(len(test.args)), a[0].GetCount())
			assertExactlyEqual(t, test.want, got)
		})
	}

}

