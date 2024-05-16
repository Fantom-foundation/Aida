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

package stochastic

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/stat/distuv"
)

// TextDeterministicNextState checks transition of a deterministic Markovian process.
func TestDeterministicNextState(t *testing.T) {
	// create random generator with fixed seed value
	rg := rand.New(rand.NewSource(999))

	var A = [][]float64{{0.0, 1.0}, {1.0, 0.0}}
	if nextState(rg, A, 0) != 1 {
		t.Fatalf("Illegal state transition (row 0)")
	}
	if nextState(rg, A, 1) != 0 {
		t.Fatalf("Illegal state transition (row 1)")
	}
}

// TextDeterministicNextState2 checks transition of a deterministic Markovian process.
func TestDeterministicNextState2(t *testing.T) {
	// create random generator with fixed seed value
	rg := rand.New(rand.NewSource(999))

	var A = [][]float64{
		{0.0, 1.0, 0.0},
		{0.0, 0.0, 1.0},
		{1.0, 0.0, 0.0},
	}
	if nextState(rg, A, 0) != 1 {
		t.Fatalf("Illegal state transition (row 0)")
	}
	if nextState(rg, A, 1) != 2 {
		t.Fatalf("Illegal state transition (row 1)")
	}
	if nextState(rg, A, 2) != 0 {
		t.Fatalf("Illegal state transition (row 1)")
	}
}

// TextNextStateFail checks whether nextState fails if
// stochastic matrix is broken.
func TestNextStateFail(t *testing.T) {
	// create random generator with fixed seed value
	rg := rand.New(rand.NewSource(999))

	var A = [][]float64{{0.0, 0.0}, {math.NaN(), 0.0}}
	if nextState(rg, A, 0) != -1 {
		t.Fatalf("Could not capture faulty stochastic matrix")
	}
	if nextState(rg, A, 1) != -1 {
		t.Fatalf("Could not capture faulty stochastic matrix")
	}
}

// checkMarkovChain checks via chi-squared test whether
// transitions are independent using the number of
// observed states. For this test, we assume that all
// rows are identical to avoid the calculation of a stationary
// distribution for an arbitrary matrix. Also the convergence
// is too slow for an arbitrary matrix.
func checkMarkovChain(A [][]float64, numSteps int) error {
	// create random generator with fixed seed value
	rg := rand.New(rand.NewSource(999))

	n := len(A)

	// number of observed states
	counts := make([]int, n)

	// run Markovian process for numSteps time
	state := 0
	for steps := 0; steps < numSteps; steps++ {
		oldState := state
		state = nextState(rg, A, state)
		if state != -1 {
			counts[state]++
		} else {
			return fmt.Errorf("State failed in step %v with outgoing probabilities of (%v)", steps, A[oldState])
		}
	}

	// compute chi-squared value for observations
	// We assume that all rows are identical.
	// For arbitrary stochastic matrix, the stationary
	// distribution must be used instead of A[0].
	chi2 := float64(0.0)
	for i, v := range counts {
		expected := float64(numSteps) * A[0][i]
		err := expected - float64(v)
		chi2 += (err * err) / expected
	}

	// Perform statistical test whether uniform Markovian process is unbiased
	// with an alpha of 0.05 and a degree of freedom of n-1 where n is the
	// number of states in the uniform Markovian process.
	alpha := 0.05
	df := float64(n - 1)
	chi2Critical := distuv.ChiSquared{K: df, Src: nil}.Quantile(1.0 - alpha)

	if chi2 > chi2Critical {
		return fmt.Errorf("Statistical test failed. Degree of freedom is %v and chi^2 value is %v; chi^2 critical value is %v", n, chi2, chi2Critical)
	}
	return nil
}

// TestRandomNextState checks whether a uniform Markovian process produces a uniform
// state distribution via a chi-squared test for various number of states.
func TestRandomNextState(t *testing.T) {
	// test small Markov chain by setting up a uniform Markovian process with
	// uniform distributions. The stationary distribution of the uniform
	// Markovian process is (1/n, , ... , 1/n).
	n := 10
	A := make([][]float64, n)
	for i := 0; i < n; i++ {
		A[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			A[i][j] = 1.0 / float64(n)
		}
	}
	if err := checkMarkovChain(A, n*n); err != nil {
		t.Fatalf("Uniform Markovian process is not unbiased for a small test-case. Error: %v", err)
	}

	// test larger uniform markov chain
	n = 5400
	A = make([][]float64, n)
	for i := 0; i < n; i++ {
		A[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			A[i][j] = 1.0 / float64(n)
		}
	}
	if err := checkMarkovChain(A, 25*n); err != nil {
		t.Fatalf("Uniform Markovian process is not unbiased for a larger test-case. Error: %v", err)
	}

	// Setup a Markovian process with a truncated geometric distributions for
	// next states. The distribution has the following formula:
	//  Pr(X=x_j) = (1-beta)*beta^n * (1-beta^n) / -beta ^ j
	// for values {x_1, ..., x_n}  of random variable X and
	// with distribution parameter beta.
	n = 10
	beta := 0.6
	A = make([][]float64, n)
	for i := 0; i < n; i++ {
		A[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			A[i][j] = ((1.0 - beta) * math.Pow(beta, float64(n)) /
				(1.0 - math.Pow(beta, float64(n)))) *
				math.Pow(beta, -float64(j+1))
		}
	}
	if err := checkMarkovChain(A, n*n); err != nil {
		t.Fatalf("Geometric Markovian process is not unbiased for a small experiment. Error: %v", err)
	}
}

// TestInitialState checks function find
// for returning the correct intial state.
func TestInitialState(t *testing.T) {
	opcode := []string{"A", "B", "C"}
	if find(opcode, "A") != 0 {
		t.Fatalf("Cannot find first state A")
	}
	if find(opcode, "B") != 1 {
		t.Fatalf("Cannot find first state B")
	}
	if find(opcode, "C") != 2 {
		t.Fatalf("Cannot find first state C")
	}
	if find(opcode, "D") != -1 {
		t.Fatalf("Should not find first state D")
	}
	if find(opcode, "") != -1 {
		t.Fatalf("Should not find first state")
	}
	if find([]string{}, "A") != -1 {
		t.Fatalf("Should not find first state A")
	}
	if find([]string{}, "") != -1 {
		t.Fatalf("Should not find first state")
	}
}
