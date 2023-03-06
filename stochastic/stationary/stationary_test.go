package stationary

import (
	"math"
	"testing"
)

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
	dist, err := ComputeDistribution(A)
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
