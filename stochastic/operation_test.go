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

package stochastic

import (
	"testing"

	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// TestOperationDecoding checks whether number encoding/decoding of operations with their arguments works.
func TestOperationDecoding(t *testing.T) {
	// enumerate whole operation space with arguments
	// and check encoding/decoding whether it is symmetric.
	for op := 0; op < NumOps; op++ {
		for addr := 0; addr < statistics.NumClasses; addr++ {
			for key := 0; key < statistics.NumClasses; key++ {
				for value := 0; value < statistics.NumClasses; value++ {
					// check legality of argument/op combination
					if (opNumArgs[op] == 0 && addr == statistics.NoArgID && key == statistics.NoArgID && value == statistics.NoArgID) ||
						(opNumArgs[op] == 1 && addr != statistics.NoArgID && key == statistics.NoArgID && value == statistics.NoArgID) ||
						(opNumArgs[op] == 2 && addr != statistics.NoArgID && key != statistics.NoArgID && value == statistics.NoArgID) ||
						(opNumArgs[op] == 3 && addr != statistics.NoArgID && key != statistics.NoArgID && value != statistics.NoArgID) {

						// encode to an argument-encoded operation
						argop := EncodeArgOp(op, addr, key, value)

						// decode argument-encoded operation
						dop, daddr, dkey, dvalue := DecodeArgOp(argop)

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
	for op := 0; op < NumOps; op++ {
		for addr := 0; addr < statistics.NumClasses; addr++ {
			for key := 0; key < statistics.NumClasses; key++ {
				for value := 0; value < statistics.NumClasses; value++ {
					// check legality of argument/op combination
					if (opNumArgs[op] == 0 && addr == statistics.NoArgID && key == statistics.NoArgID && value == statistics.NoArgID) ||
						(opNumArgs[op] == 1 && addr != statistics.NoArgID && key == statistics.NoArgID && value == statistics.NoArgID) ||
						(opNumArgs[op] == 2 && addr != statistics.NoArgID && key != statistics.NoArgID && value == statistics.NoArgID) ||
						(opNumArgs[op] == 3 && addr != statistics.NoArgID && key != statistics.NoArgID && value != statistics.NoArgID) {

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
