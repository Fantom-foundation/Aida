package stochastic

import (
	"testing"
)

// TestOperationDecoding checks whether number encoding/decoding of operations with their arguments works.
func TestNextState(t *testing.T) {
	var A = [][]float64{{0.0, 1.0}, {1.0, 0.0}}
	i := nextState(A, 0)
	if i != 1 {
		t.Fatalf("Illegal state transition (row 0)")
	}
	i = nextState(A, 1)
	if i != 0 {
		t.Fatalf("Illegal state transition (row 1)")
	}
}

// TextNextStateFail checks whether Markov property holds.
func TestNextStateFail(t *testing.T) {
	var A = [][]float64{{0.0, 0.0}, {0.0, 0.0}}
	i := nextState(A, 0)
	if i != -1 {
		t.Fatalf("Could not capture faulty stochastic matrix")
	}
}
