package stochastic

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestEventRegistryUpdateFreq checks some operation labels with their argument classes.
func TestEventRegistryUpdateFreq(t *testing.T) {
	r := NewEventRegistry()

	// check that frequencies of argument-encoded operations and
	// transit frequencies are zero.
	for i := 0; i < numArgOps; i++ {
		if r.argOpFreq[i] > 0 {
			t.Fatalf("Operation frequency must be zero")
		}
		for j := 0; j < numArgOps; j++ {
			if r.transitFreq[i][j] > 0 {
				t.Fatalf("Transit frequency must be zero")
			}
		}
	}

	// inject first operation
	op := CreateAccountID
	addr := randomValueID
	key := noArgID
	value := noArgID
	r.updateFreq(op, addr, key, value)
	argop1 := EncodeArgOp(op, addr, key, value)

	// check updated operation/transit frequencies
	for i := 0; i < numArgOps; i++ {
		for j := 0; j < numArgOps; j++ {
			if r.transitFreq[i][j] > 0 {
				t.Fatalf("Transit frequency must be zero")
			}
		}
		if i != argop1 && r.argOpFreq[i] > 0 {
			t.Fatalf("Operation frequency must be zero")
		}
	}
	if r.argOpFreq[argop1] != 1 {
		t.Fatalf("Operation frequency must be one")
	}

	// inject second operation
	op = SetStateID
	addr = randomValueID
	key = previousValueID
	value = zeroValueID
	r.updateFreq(op, addr, key, value)
	argop2 := EncodeArgOp(op, addr, key, value)
	for i := 0; i < numArgOps; i++ {
		for j := 0; j < numArgOps; j++ {
			if r.transitFreq[i][j] > 0 && i != argop1 && j != argop2 {
				t.Fatalf("Transit frequency must be zero")
			}
		}
	}
	for i := 0; i < numArgOps; i++ {
		if (i == argop1 || i == argop2) && r.argOpFreq[i] != 1 {
			t.Fatalf("Operation frequency must be one")
		}
		if (i != argop1 && i != argop2) && r.argOpFreq[i] > 0 {
			t.Fatalf("Operation frequency must be zero")
		}
	}
	if r.transitFreq[argop1][argop2] != 1 {
		t.Fatalf("Transit frequency must be one %v", r.transitFreq[argop2][argop1])
	}
}

// check frequencies
func checkFrequencies(r *EventRegistry, opFreq [numArgOps]uint64, transitFreq [numArgOps][numArgOps]uint64) bool {
	for i := 0; i < numArgOps; i++ {
		if r.argOpFreq[i] != opFreq[i] {
			return false
		}
		for j := 0; j < numArgOps; j++ {
			if r.transitFreq[i][j] != transitFreq[i][j] {
				return false
			}
		}
	}
	return true
}

// TestEventRegistryOperation checks registration for operations
func TestEventRegistryOperation(t *testing.T) {
	// operation/transit frequencies
	var (
		opFreq      [numArgOps]uint64
		transitFreq [numArgOps][numArgOps]uint64
	)

	// create new event registry
	r := NewEventRegistry()

	// check that frequencies are zero.
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject first operation and check frequencies.
	addr := common.HexToAddress("0x000000010")
	r.RegisterAddressOp(CreateAccountID, &addr)
	argop1 := EncodeArgOp(CreateAccountID, newValueID, noArgID, noArgID)
	opFreq[argop1]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject second operation and check frequencies.
	key := common.HexToHash("0x000000200")
	r.RegisterKeyOp(GetStateID, &addr, &key)
	argop2 := EncodeArgOp(GetStateID, previousValueID, newValueID, noArgID)
	opFreq[argop2]++
	transitFreq[argop1][argop2]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject third operation and check frequencies.
	value := common.Hash{}
	r.RegisterValueOp(SetStateID, &addr, &key, &value)
	argop3 := EncodeArgOp(SetStateID, previousValueID, previousValueID, zeroValueID)
	opFreq[argop3]++
	transitFreq[argop2][argop3]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject forth operation and check frequencies.
	r.RegisterOp(SnapshotID)
	argop4 := EncodeArgOp(SnapshotID, noArgID, noArgID, noArgID)
	opFreq[argop4]++
	transitFreq[argop3][argop4]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}
}

// TestEventRegistryZeroOperation checks zero value, new and previous argument classes.
func TestEventRegistryZeroOperation(t *testing.T) {
	// operation/transit frequencies
	var (
		opFreq      [numArgOps]uint64
		transitFreq [numArgOps][numArgOps]uint64
	)

	// create new event registry
	r := NewEventRegistry()

	// check that frequencies are zero.
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject first operation and check frequencies.
	addr := common.Address{}
	key := common.Hash{}
	value := common.Hash{}
	r.RegisterValueOp(SetStateID, &addr, &key, &value)
	argop1 := EncodeArgOp(SetStateID, zeroValueID, zeroValueID, zeroValueID)
	opFreq[argop1]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject second operation and check frequencies.
	addr = common.HexToAddress("0x12312121212")
	key = common.HexToHash("0x232313123123213")
	value = common.HexToHash("0x2301238021830912830")
	r.RegisterValueOp(SetStateID, &addr, &key, &value)
	argop2 := EncodeArgOp(SetStateID, newValueID, newValueID, newValueID)
	opFreq[argop2]++
	transitFreq[argop1][argop2]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject third operation and check frequencies.
	r.RegisterValueOp(SetStateID, &addr, &key, &value)
	argop3 := EncodeArgOp(SetStateID, previousValueID, previousValueID, previousValueID)
	opFreq[argop3]++
	transitFreq[argop2][argop3]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}
}
