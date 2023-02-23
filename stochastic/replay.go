package stochastic

import (
	"fmt"
	"log"
	"math/rand"
)

// RunStochasticReplay runs the stochastic simulation for StateDB operations.
// It requires the simulation model and simulation length. The verbose enables/disables
// the printing of StateDB operations and their arguments on the screen.
func RunStochasticReplay(e *EstimationModelJSON, simLength int, verbose bool) {
	opcode := e.Operations
	A := e.StochasticMatrix

	// produce random access generators for contract addresses,
	// storage-keys, and storage addresses.

	// Contracts need an indirect access wrapper because
	// contract addresses can be deleted by suicide.
	contracts := NewIndirectAccess(NewRandomAccess(
		e.Contracts.NumKeys,
		e.Contracts.Lambda,
		e.Contracts.QueueDistribution,
	))
	keys := NewRandomAccess(
		e.Keys.NumKeys,
		e.Keys.Lambda,
		e.Keys.QueueDistribution,
	)
	values := NewRandomAccess(
		e.Values.NumKeys,
		e.Values.Lambda,
		e.Values.QueueDistribution,
	)

	// set initial state (ensure that zero is a start-state)
	state := initialState(opcode, "SN")
	if state == -1 {
		log.Fatalf("Initial state cannot be observed in Markov chain/recording failed.")
	}

	for i := 0; i < simLength; i++ {

		// decode opcode
		op, addrCl, keyCl, valueCl := DecodeOpcode(opcode[state])

		// Fetch random indexes from random access generators
		addrIdx := contracts.NextIndex(addrCl)
		keyIdx := keys.NextIndex(keyCl)
		valueIdx := values.NextIndex(valueCl)

		// print opcode and its randomly generated index parameters.
		if verbose {
			fmt.Printf("opcode:%v", opcode[state])
			if addrCl != noArgID {
				fmt.Printf(" %v", addrIdx)
			}
			if keyCl != noArgID {
				fmt.Printf(" %v", keyIdx)
			}
			if valueCl != noArgID {
				fmt.Printf(" %v", valueIdx)
			}
			fmt.Println()
		}

		// execute state
		executeState(op, addrIdx, keyIdx, valueIdx)

		// transit to next state
		state = nextState(A, state)
	}
}

// executeState executes StateDB operation
func executeState(op int, addr, key, value int64) {
	// 1) convert indices to 20-byte/ 32-byte addresses/hashes
	// 2) issue stateDB operations (for the execution we may need some state, e.g., balance, etc)
}

// initialState returns the row/column index of the first state in the stochastic matrix.
func initialState(operations []string, opcode string) int {
	for i, opc := range operations {
		if opc == opcode {
			return i
		}
	}
	return -1
}

// nextState produces the next state in the Markovian process.
func nextState(A [][]float64, i int) int {
	// Retrieve a random number in [0,1.0).
	r := rand.Float64()

	// Use Kahan's sum for summing values
	// in case we have a combination of very small
	// and very large values.
	sum := float64(0.0)
	c := float64(0.0)
	k := -1
	for j := 0; j < len(A); j++ {
		y := A[i][j] - c
		t := sum + y
		c = (t - sum) - y
		sum = t
		if r <= sum {
			return j
		}
		// If we have a numerical unstable cumulative
		// distribution (large and small numbers that cancel
		// each other out when summing up), we can take the last
		// non-zero entry as a solution. It also detects
		// stochastic matrices with a row whose row
		// sum is not zero (return value is -1 for such a case).
		if A[i][j] > 0.0 {
			k = j
		}
	}
	return k
}
