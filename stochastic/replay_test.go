package stochastic

import (
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/stat/distuv"
)

// TextNextState checks transition of a deterministic Markovian process.
func TestNextState(t *testing.T) {
	var A = [][]float64{{0.0, 1.0}, {1.0, 0.0}}
	if nextState(A, 0) != 1 {
		t.Fatalf("Illegal state transition (row 0)")
	}
	if nextState(A, 1) != 0 {
		t.Fatalf("Illegal state transition (row 1)")
	}
}

// TextNextState2 checks transition of a deterministic Markovian process.
func TestNextState2(t *testing.T) {
	var A = [][]float64{
		{0.0, 1.0, 0.0},
		{0.0, 0.0, 1.0},
		{1.0, 0.0, 0.0},
	}
	i := nextState(A, 0)
	if i != 1 {
		t.Fatalf("Illegal state transition (row 0)")
	}
	i = nextState(A, 1)
	if i != 2 {
		t.Fatalf("Illegal state transition (row 1)")
	}
	i = nextState(A, 2)
	if i != 0 {
		t.Fatalf("Illegal state transition (row 1)")
	}
}

// TextNextStateFail checks whether nextState fails if Markov property does not hold.
func TestNextStateFail(t *testing.T) {
	var A = [][]float64{{0.0, 0.0}, {0.0, 0.0}}
	if nextState(A, 0) != -1 {
		t.Fatalf("Could not capture faulty stochastic matrix")
	}
}

// checkUniformMarkov checks via chi-squared test whether
// transitions are truly independent using the number of
// observed states.
func checkUniformMarkov(n int, numSteps int) bool {

	// setup uniform Markovian process with
	// uniform distributions. The stationary distribution
	// of the uniform Markovian process is
	// (1/n, , ... , 1/n)
	A := make([][]float64, n)
	for i := 0; i < n; i++ {
		A[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			A[i][j] = 1.0 / float64(n)
		}
	}

	// number of observed states
	counts := make([]int, n)

	// run Markovian process for numSteps time
	state := 0
	for steps := 0; steps < numSteps; steps++ {
		state = nextState(A, state)
		counts[state]++
	}

	// compute chi-squared value for observations
	chi2 := float64(0.0)
	expected := float64(numSteps) / float64(n)
	for _, v := range counts {
		err := expected - float64(v)
		// fmt.Printf("Err: %v %v\n", v, expected)
		chi2 += (err * err) / expected
	}

	// Perform statistical test whether uniform Markovian process is unbiased
	// with an alpha of 0.05 and a degree of freedom of n-1 where n is the
	// number of states in the uniform Markovian process.
	alpha := 0.05
	df := float64(n - 1)
	chi2Critical := distuv.ChiSquared{K: df, Src: nil}.Quantile(1.0 - alpha)
	// fmt.Printf("Chi^2 value: %v Chi^2 critical value: %v n: %v\n", chi2, chi2Critical, n)

	return chi2 <= chi2Critical
}

// TestRandomNextState checks whether a uniform Markovian process
// produces a uniform state distribution via a chi-squared test
// for various number of states.
func TestRandomNextState(t *testing.T) {
	// set random seed to make test deterministic
	// (make sure that these tests are not performed in parallel)
	rand.Seed(4711)

	// test small markov chain
	if !checkUniformMarkov(4, 100) {
		t.Fatalf("Uniform Markovian process is not unbiased for small test.")
	}

	// test large markov chain
	if !checkUniformMarkov(5400, 25*5400) {
		t.Fatalf("Uniform Markovian process is not unbiased for large test.")
	}
}

// TestInitialState checks function initialState
// for returning the correct intial state.
func TestInitialState(t *testing.T) {
	opcode := []string{"A", "B", "C"}
	if initialState(opcode, "A") != 0 {
		t.Fatalf("Cannot find first state A")
	}
	if initialState(opcode, "B") != 1 {
		t.Fatalf("Cannot find first state B")
	}
	if initialState(opcode, "C") != 2 {
		t.Fatalf("Cannot find first state C")
	}
	if initialState(opcode, "D") != -1 {
		t.Fatalf("Should not find first state D")
	}
	if initialState(opcode, "") != -1 {
		t.Fatalf("Should not find first state")
	}
	if initialState([]string{}, "A") != -1 {
		t.Fatalf("Should not find first state A")
	}
	if initialState([]string{}, "") != -1 {
		t.Fatalf("Should not find first state")
	}
}
