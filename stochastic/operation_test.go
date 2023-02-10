package stochastic

import (
	"testing"
)

// TestOperationDecoding checks whether number encoding/decoding of operations with their arguments works.
func TestOperationDecoding(t *testing.T) {
	// enumerate whole operation space with arguments
	// and check encoding/decoding whether it is symmetric.
	for op := 0; op < numOps; op++ {
		for addr := 0; addr < numClasses; addr++ {
			for key := 0; key < numClasses; key++ {
				for value := 0; value < numClasses; value++ {
					// check legality of argument/op combination
					if (opNumArgs[op] == 0 && addr == noArgID && key == noArgID && value == noArgID) ||
						(opNumArgs[op] == 1 && addr != noArgID && key == noArgID && value == noArgID) ||
						(opNumArgs[op] == 2 && addr != noArgID && key != noArgID && value == noArgID) ||
						(opNumArgs[op] == 3 && addr != noArgID && key != noArgID && value != noArgID) {

						// encode to an argument-encoded operation
						argop := EncodeArgOp(op, addr, key, value)

						// decode argument-encoded operation
						dop, daddr, dkey, dvalue := decodeArgOp(argop)

						if op != dop || addr != daddr || key != dkey || value != dvalue {
							t.Fatalf("Encoding/decoding failed")
						}
					}
				}
			}
		}
	}
}

// TestOperationOpcode checks the mnemonic encoding/decoding of operations with their argument classes as opcode.
func TestOperationOpcode(t *testing.T) {
	// enumerate whole operation space with arguments
	// and check encoding/decoding whether it is symmetric.
	for op := 0; op < numOps; op++ {
		for addr := 0; addr < numClasses; addr++ {
			for key := 0; key < numClasses; key++ {
				for value := 0; value < numClasses; value++ {
					// check legality of argument/op combination
					if (opNumArgs[op] == 0 && addr == noArgID && key == noArgID && value == noArgID) ||
						(opNumArgs[op] == 1 && addr != noArgID && key == noArgID && value == noArgID) ||
						(opNumArgs[op] == 2 && addr != noArgID && key != noArgID && value == noArgID) ||
						(opNumArgs[op] == 3 && addr != noArgID && key != noArgID && value != noArgID) {

						// encode to an argument-encoded operation
						opcode := EncodeOpcode(op, addr, key, value)

						// decode argument-encoded operation
						dop, daddr, dkey, dvalue := DecodeOpcode(opcode)

						if op != dop || addr != daddr || key != dkey || value != dvalue {
							t.Fatalf("Encoding/decoding failed for %v", opcode)
						}
					}
				}
			}
		}
	}
}
