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
	"log"

	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// IDs of StateDB Operations
const (
	AddBalanceID = iota
	BeginBlockID
	BeginSyncPeriodID
	BeginTransactionID
	CreateAccountID
	EmptyID
	EndBlockID
	EndSyncPeriodID
	EndTransactionID
	ExistID
	GetBalanceID
	GetCodeHashID
	GetCodeID
	GetCodeSizeID
	GetCommittedStateID
	GetNonceID
	GetStateID
	HasSuicidedID
	RevertToSnapshotID
	SetCodeID
	SetNonceID
	SetStateID
	SnapshotID
	SubBalanceID
	SuicideID
	GetTransientStateID
	SetTransientStateID

	NumOps
)

// numArgOps gives the number of operations with encoded argument classes
const numArgOps = NumOps * statistics.NumClasses * statistics.NumClasses * statistics.NumClasses

// opText translates IDs to operation's text
var opText = map[int]string{
	AddBalanceID:        "AddBalance",
	BeginBlockID:        "BeginBlock",
	BeginSyncPeriodID:   "BeginSyncPeriod",
	BeginTransactionID:  "BeginTransaction",
	CreateAccountID:     "CreateAccount",
	EmptyID:             "Empty",
	EndBlockID:          "EndBlock",
	EndSyncPeriodID:     "EndSyncPeriod",
	EndTransactionID:    "EndTransaction",
	ExistID:             "Exist",
	GetBalanceID:        "GetBalance",
	GetCodeHashID:       "GetCodeHash",
	GetCodeID:           "GetCode",
	GetCodeSizeID:       "GetCodeSize",
	GetCommittedStateID: "GetCommittedState",
	GetNonceID:          "GetNonce",
	GetStateID:          "GetState",
	HasSuicidedID:       "HasSuicided",
	RevertToSnapshotID:  "RevertToSnapshot",
	SetCodeID:           "SetCode",
	SetNonceID:          "SetNonce",
	SetStateID:          "SetState",
	SnapshotID:          "Snapshot",
	SubBalanceID:        "SubBalance",
	SuicideID:           "Suicide",
	GetTransientStateID: "GetTransientState",
	SetTransientStateID: "SetTransientState",
}

// opMnemo is a mnemonics table for operations.
var opMnemo = map[int]string{
	AddBalanceID:        "AB",
	BeginBlockID:        "BB",
	BeginSyncPeriodID:   "BS",
	BeginTransactionID:  "BT",
	CreateAccountID:     "CA",
	EmptyID:             "EM",
	EndBlockID:          "EB",
	EndSyncPeriodID:     "ES",
	EndTransactionID:    "ET",
	ExistID:             "EX",
	GetBalanceID:        "GB",
	GetCodeHashID:       "GH",
	GetCodeID:           "GC",
	GetCodeSizeID:       "GZ",
	GetCommittedStateID: "GM",
	GetNonceID:          "GN",
	GetStateID:          "GS",
	HasSuicidedID:       "HS",
	RevertToSnapshotID:  "RS",
	SetCodeID:           "SC",
	SetNonceID:          "SO",
	SetStateID:          "SS",
	SnapshotID:          "SN",
	SubBalanceID:        "SB",
	SuicideID:           "SU",
	GetTransientStateID: "GT",
	SetTransientStateID: "ST",
}

// opNumArgs is an argument number table for operations.
var opNumArgs = map[int]int{
	AddBalanceID:        1,
	BeginBlockID:        0,
	BeginSyncPeriodID:   0,
	BeginTransactionID:  0,
	CreateAccountID:     1,
	EmptyID:             1,
	EndBlockID:          0,
	EndSyncPeriodID:     0,
	EndTransactionID:    0,
	ExistID:             1,
	GetBalanceID:        1,
	GetCodeHashID:       1,
	GetCodeID:           1,
	GetCodeSizeID:       1,
	GetCommittedStateID: 2,
	GetNonceID:          1,
	GetStateID:          2,
	HasSuicidedID:       1,
	RevertToSnapshotID:  0,
	SetCodeID:           1,
	SetNonceID:          1,
	SetStateID:          3,
	SnapshotID:          0,
	SubBalanceID:        1,
	SuicideID:           1,
	GetTransientStateID: 2,
	SetTransientStateID: 3,
}

// opId is an operation ID table.
var opId = map[string]int{
	"AB": AddBalanceID,
	"BB": BeginBlockID,
	"BS": BeginSyncPeriodID,
	"BT": BeginTransactionID,
	"CA": CreateAccountID,
	"EM": EmptyID,
	"EB": EndBlockID,
	"ES": EndSyncPeriodID,
	"ET": EndTransactionID,
	"EX": ExistID,
	"GB": GetBalanceID,
	"GH": GetCodeHashID,
	"GC": GetCodeID,
	"GZ": GetCodeSizeID,
	"GM": GetCommittedStateID,
	"GN": GetNonceID,
	"GS": GetStateID,
	"HS": HasSuicidedID,
	"RS": RevertToSnapshotID,
	"SC": SetCodeID,
	"SO": SetNonceID,
	"SN": SnapshotID,
	"SB": SubBalanceID,
	"SS": SetStateID,
	"SU": SuicideID,
	"ST": SetTransientStateID,
	"GT": GetTransientStateID,
}

// argMnemo is the argument-class mnemonics table.
var argMnemo = map[int]string{
	statistics.NoArgID:         "",
	statistics.ZeroValueID:     "z",
	statistics.NewValueID:      "n",
	statistics.PreviousValueID: "p",
	statistics.RecentValueID:   "q",
	statistics.RandomValueID:   "r",
}

// argId is the argument-class id table.
var argId = map[byte]int{
	'z': statistics.ZeroValueID,
	'n': statistics.NewValueID,
	'p': statistics.PreviousValueID,
	'q': statistics.RecentValueID,
	'r': statistics.RandomValueID,
}

// OpMnemo returns the mnemonic code for an operation.
func OpMnemo(op int) string {
	if op < 0 || op >= NumOps {
		panic("opcode is out of range")
	}
	return opMnemo[op]
}

// checkArgOp checks whether op/argument combination is valid.
func checkArgOp(op int, contract int, key int, value int) bool {
	if op < 0 || op >= NumOps {
		return false
	}
	if contract < 0 || contract >= statistics.NumClasses {
		return false
	}
	if key < 0 || key >= statistics.NumClasses {
		return false
	}
	if value < 0 || value >= statistics.NumClasses {
		return false
	}
	switch opNumArgs[op] {
	case 0:
		return contract == statistics.NoArgID &&
			key == statistics.NoArgID &&
			value == statistics.NoArgID
	case 1:
		return contract != statistics.NoArgID &&
			key == statistics.NoArgID &&
			value == statistics.NoArgID
	case 2:
		return contract != statistics.NoArgID &&
			key != statistics.NoArgID &&
			value == statistics.NoArgID
	case 3:
		return contract != statistics.NoArgID &&
			key != statistics.NoArgID &&
			value != statistics.NoArgID
	default:
		return false
	}
}

// IsValidArgOp returns true if the encoding is valid.
func IsValidArgOp(argop int) bool {
	if argop < 0 || argop >= numArgOps {
		return false
	}
	op, contract, key, value := DecodeArgOp(argop)
	return checkArgOp(op, contract, key, value)
}

// EncodeArgOp encodes operation and argument classes via Horner's scheme to a single value.
func EncodeArgOp(op int, addr int, key int, value int) int {
	if !checkArgOp(op, addr, key, value) {
		log.Fatalf("EncodeArgOp: invalid operation/arguments")
	}
	return (((int(op)*statistics.NumClasses)+addr)*statistics.NumClasses+key)*statistics.NumClasses + value
}

// DecodeArgOp decodes operation with arguments using Honer's scheme
func DecodeArgOp(argop int) (int, int, int, int) {
	if argop < 0 || argop >= numArgOps {
		log.Fatalf("DecodeArgOp: invalid op range")
	}

	value := argop % statistics.NumClasses
	argop = argop / statistics.NumClasses

	key := argop % statistics.NumClasses
	argop = argop / statistics.NumClasses

	addr := argop % statistics.NumClasses
	argop = argop / statistics.NumClasses

	op := argop

	return op, addr, key, value
}

// EncodeOpcode generates the opcode for an operation and its argument classes.
func EncodeOpcode(op int, addr int, key int, value int) string {
	if !checkArgOp(op, addr, key, value) {
		log.Fatalf("EncodeOpcode: invalid operation/arguments")
	}
	code := fmt.Sprintf("%v%v%v%v", opMnemo[op], argMnemo[addr], argMnemo[key], argMnemo[value])
	if len(code) != 2+opNumArgs[op] {
		log.Fatalf("EncodeOpcode: wrong opcode length for opcode %v", code)
	}
	return code
}

// validateArg checks whether argument mnemonics exists.
func validateArg(argMnemo byte) bool {
	_, ok := argId[argMnemo]
	return ok
}

// DecodeOpcode decodes opcode producing the operation id and its argument classes
func DecodeOpcode(opc string) (int, int, int, int) {
	mnemo := opc[:2]
	op, ok := opId[mnemo]
	if !ok {
		log.Fatalf("DecodeOpcode: lookup failed for %v", mnemo)
	}
	if len(opc) != 2+opNumArgs[op] {
		log.Fatalf("DecodeOpcode: wrong opcode length for %v", opc)
	}
	var contract, key, value int
	switch len(opc) - 2 {
	case 0:
		contract, key, value = statistics.NoArgID, statistics.NoArgID, statistics.NoArgID
	case 1:
		if !validateArg(opc[2]) {
			log.Fatalf("DecodeOpcode: wrong argument code")
		}
		contract, key, value = argId[opc[2]], statistics.NoArgID, statistics.NoArgID
	case 2:
		if !validateArg(opc[2]) || !validateArg(opc[3]) {
			log.Fatalf("DecodeOpcode: wrong argument code")
		}
		contract, key, value = argId[opc[2]], argId[opc[3]], statistics.NoArgID
	case 3:
		if !validateArg(opc[2]) || !validateArg(opc[3]) || !validateArg(opc[4]) {
			log.Fatalf("DecodeOpcode: wrong argument code")
		}
		contract, key, value = argId[opc[2]], argId[opc[3]], argId[opc[4]]
	}
	return op, contract, key, value
}
