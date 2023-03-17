package exponential

import (
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/stat/distuv"
)

// TODO: pointwise tests of CDF/Quantile with a list of known points (see gonum package)

// TestEstimation checks the correcntness of approximating a lambda for a discrete CDF.
func TestEstimation(t *testing.T) {
	for l := 2; l < 900; l += 5 {
		checkEstimation(t, float64(l))
	}
}

// checkEstimation checks whether the approximate lambda can be
// rediscovered from a discretized CDF.
func checkEstimation(t *testing.T, expectedLambda float64) {
	Cdf := PiecewiseLinearCdf(expectedLambda, 100)
	computedLambda, err := ApproximateLambda(Cdf)
	if err != nil {
		t.Fatalf("Failed to approximate. Error: %v", err)
	}
	if math.Abs(expectedLambda-computedLambda) > estimationEps {
		t.Fatalf("Failed to approximate. Expected Lambda:%v Computed Lambda: %v", expectedLambda, computedLambda)
	}
}

// TestRandomAccessRandInd checks the random selection of the queue position via a statistical test.
func TestRandomAccessRandInd(t *testing.T) {
	// create random generator with fixed seed value
	rg := rand.New(rand.NewSource(999))

	// parameters
	lambda := 5.0
	numSteps := 10000
	idxRange := int64(10)

	// populate buckets
	counts := make([]int64, idxRange)
	for steps := 0; steps < numSteps; steps++ {
		counts[DiscreteSample(rg, lambda, idxRange)]++
	}

	// compute chi-squared value for observations
	chi2 := float64(0.0)
	for i, v := range counts {
		// compute expected value of bucket
		p := Cdf(lambda, float64(i+1)/float64(idxRange)) - Cdf(lambda, float64(i)/float64(idxRange))
		expected := float64(numSteps) * p
		err := expected - float64(v)
		chi2 += (err * err) / expected
		// fmt.Printf("Err: %v %v\n", v, expected)
	}

	// Perform statistical test whether uniform queue distribution is unbiased
	// with an alpha of 0.05 and a degree of freedom of queue length minus two
	// (no first position!).
	alpha := 0.05
	df := float64(idxRange - 1)
	chi2Critical := distuv.ChiSquared{K: df, Src: nil}.Quantile(1.0 - alpha)
	// fmt.Printf("Chi^2 value: %v Chi^2 critical value: %v df: %v\n", chi2, chi2Critical, df)

	if chi2 > chi2Critical {
		t.Fatalf("The random index selection biased.")
	}
}
