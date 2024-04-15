// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
