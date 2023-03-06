package exponential

import (
	"math"
	"testing"
)

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
