package stochastic

import (
	"math"
	"testing"
)

// TestEstimation checks the correcntness of approximating
// a lambda for a discrete CDF.
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

// checkStationaryDistribution tests stationary distribution of a uniform Markovian process
// whose transition probability is 1/n for n states.
func checkStationaryDistribution(t *testing.T, n int) {
	A := make([][]float64, n)
	for i := 0; i < n; i++ {
		A[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			A[i][j] = 1.0 / float64(n)
		}
	}
	eps := 1e-3
	dist, err := StationaryDistribution(A)
	if err != nil {
		t.Fatalf("Failed to compute stationary distribution. Error: %v", err)
	}
	for i := 0; i < n; i++ {
		if dist[i] < 0.0 || dist[i] > 1.0 {
			t.Fatalf("Not a probability in distribution.")
		}
		if math.Abs(dist[i]-1.0/float64(n)) > eps {
			t.Fatalf("Failed to compute sufficiently precise stationary distribution.")
		}
	}
}

// TestStationaryDistribution of a Markov Chain
// TestEstimation checks the correcntness of approximating
// a lambda for a discrete CDF.
func TestStationaryDistribution(t *testing.T) {
	for n := 2; n < 10; n++ {
		checkStationaryDistribution(t, n)
	}
}
