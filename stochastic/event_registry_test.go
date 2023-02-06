package stochastic

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestEventRegistryDecoding checks whether encoding/decoding of operations with their arguments works.
func TestEventRegistryDecoding(t *testing.T) {
	// enumerate whole operation space with arguments
	// and check encoding/decoding whether it is symmetric.
	for op := 0; op < numOps; op++ {
		for addr := 0; addr < numClasses; addr++ {
			for key := 0; key < numClasses; key++ {
				for value := 0; value < numClasses; value++ {
					argop := encodeOp(op, addr, key, value)
					dop, daddr, dkey, dvalue := decodeOp(argop)
					if op != dop || addr != daddr || key != dkey || value != dvalue {
						t.Fatalf("Encoding/decoding failed")
					}
				}
			}
		}
	}
}

// TestEventRegistryLabel checks some operation labels with their argument classes.
func TestEventRegistryLabel(t *testing.T) {
	txt := opLabel(snapshotID, noArgEntry, noArgEntry, noArgEntry)
	if txt != "SN" {
		t.Fatalf("Wrong operation label for SN")
	}

	txt = opLabel(getStateID, randomEntry, previousEntry, noArgEntry)
	if txt != "GSrp" {
		t.Fatalf("Wrong operation label for GSrp")
	}

	txt = opLabel(setStateID, zeroEntry, previousEntry, randomEntry)
	if txt != "SSzpr" {
		t.Fatalf("Wrong operation label for SSzpr")
	}

	txt = opLabel(setStateID, randomEntry, zeroEntry, recentEntry)
	if txt != "SSrzq" {
		t.Fatalf("Wrong operation label for SSrzq")
	}
}

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
	op := createAccountID
	addr := randomEntry
	key := noArgEntry
	value := noArgEntry
	r.updateFreq(op, addr, key, value)
	argop1 := encodeOp(op, addr, key, value)

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
	op = setStateID
	addr = randomEntry
	key = previousEntry
	value = zeroEntry
	r.updateFreq(op, addr, key, value)
	argop2 := encodeOp(op, addr, key, value)
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

// TestEventRegistryLabel checks registration
// TODO: have more of similar tests with previous/recent address/key/value
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
	r.RegisterAddressOp(createAccountID, &addr)
	argop1 := encodeOp(createAccountID, newEntry, noArgEntry, noArgEntry)
	opFreq[argop1]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject second operation and check frequencies.
	key := common.HexToHash("0x000000200")
	r.RegisterKeyOp(getStateID, &addr, &key)
	argop2 := encodeOp(getStateID, previousEntry, newEntry, noArgEntry)
	opFreq[argop2]++
	transitFreq[argop1][argop2]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject third operation and check frequencies.
	value := common.Hash{}
	r.RegisterValueOp(setStateID, &addr, &key, &value)
	argop3 := encodeOp(setStateID, previousEntry, previousEntry, zeroEntry)
	opFreq[argop3]++
	transitFreq[argop2][argop3]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}

	// inject forth operation and check frequencies.
	r.RegisterOp(snapshotID)
	argop4 := encodeOp(snapshotID, noArgEntry, noArgEntry, noArgEntry)
	opFreq[argop4]++
	transitFreq[argop3][argop4]++
	if !checkFrequencies(&r, opFreq, transitFreq) {
		t.Fatalf("operation/transit frequency diverges")
	}
}
