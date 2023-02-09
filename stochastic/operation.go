package stochastic

import (
	"fmt"
	"log"
)

// IDs of StateDB Operations
const (
	addBalanceID = iota
	createAccountID
	emptyID
	existID
	finaliseID
	getBalanceID
	getCodeHashID
	getCodeID
	getCodeSizeID
	getCommittedStateID
	getNonceID
	getStateID
	hasSuicidedID
	revertToSnapshotID
	setCodeID
	setNonceID
	setStateID
	snapshotID
	subBalanceID
	suicideID

	numOps
)

// IDs for argument classes
const (
	noArgID         = iota // default label (for no argument)
	zeroValueID            // zero value access
	newValueID             // newly occurring value access
	previousValueID        // value that was previously accessed
	recentValueID          // value that recently accessed (time-window is fixed to qstatsLen)
	randomValueID          // random access (everything else)

	numClasses
)

// opMnemo is a mnemonics table for operations.
var opMnemo = map[int]string{
	addBalanceID:        "AB",
	createAccountID:     "CA",
	emptyID:             "EM",
	existID:             "EX",
	finaliseID:          "FI",
	getBalanceID:        "GB",
	getCodeHashID:       "GH",
	getCodeID:           "GC",
	getCodeSizeID:       "GZ",
	getCommittedStateID: "GM",
	getNonceID:          "GN",
	getStateID:          "GS",
	hasSuicidedID:       "HS",
	revertToSnapshotID:  "RS",
	setCodeID:           "SC",
	setNonceID:          "SO",
	snapshotID:          "SN",
	subBalanceID:        "SB",
	setStateID:          "SS",
	suicideID:           "SU",
}

// opNumArgs is an argument number table for operations.
var opNumArgs = map[int]int{
	addBalanceID:        1,
	createAccountID:     1,
	emptyID:             1,
	existID:             1,
	finaliseID:          0,
	getBalanceID:        1,
	getCodeHashID:       1,
	getCodeID:           1,
	getCodeSizeID:       1,
	getCommittedStateID: 2,
	getNonceID:          1,
	getStateID:          2,
	hasSuicidedID:       1,
	revertToSnapshotID:  0,
	setCodeID:           1,
	setNonceID:          1,
	snapshotID:          0,
	subBalanceID:        1,
	setStateID:          3,
	suicideID:           1,
}

// opId is an operation ID table.
var opId = map[string]int{
	"AB": addBalanceID,
	"CA": createAccountID,
	"EM": emptyID,
	"EX": existID,
	"FI": finaliseID,
	"GB": getBalanceID,
	"GH": getCodeHashID,
	"GC": getCodeID,
	"GZ": getCodeSizeID,
	"GM": getCommittedStateID,
	"GN": getNonceID,
	"GS": getStateID,
	"HS": hasSuicidedID,
	"RS": revertToSnapshotID,
	"SC": setCodeID,
	"SO": setNonceID,
	"SN": snapshotID,
	"SB": subBalanceID,
	"SS": setStateID,
	"SU": suicideID,
}

// argMnemo is the argument-class mnemonics table.
var argMnemo = map[int]string{
	noArgID:         "",
	zeroValueID:     "z",
	newValueID:      "n",
	previousValueID: "p",
	recentValueID:   "q",
	randomValueID:   "r",
}

// argId is the argument-class id table.
var argId = map[byte]int{
	'z': zeroValueID,
	'n': newValueID,
	'p': previousValueID,
	'q': recentValueID,
	'r': randomValueID,
}

// checkArgOp checks whether op/argument combination is valid.
func checkArgOp(op int, addr int, key int, value int) bool {
	if op < 0 || op >= numOps {
		return false
	}
	if addr < 0 || addr >= numClasses {
		return false
	}
	if (opNumArgs[op] >= 1 && addr == noArgID) || (opNumArgs[op] == 0 && addr != noArgID) {
		return false
	}
	if key < 0 || key >= numClasses {
		return false
	}
	if (opNumArgs[op] >= 2 && key == noArgID) || (opNumArgs[op] <= 1 && key != noArgID) {
		return false
	}
	if value < 0 || value >= numClasses {
		return false
	}
	if (opNumArgs[op] == 3 && value == noArgID) || (opNumArgs[op] <= 2 && value != noArgID) {
		return false
	}
	return true
}

// EncodeOp encodes operation and argument classes via Horner's scheme to a single value.
func EncodeArgOp(op int, addr int, key int, value int) int {
	if !checkArgOp(op, addr, key, value) {
		log.Fatalf("EncodeArgOp: invalid operation/arguments")
	}
	return (((int(op)*numClasses)+addr)*numClasses+key)*numClasses + value
}

// DecodeOp decodes operation with arguments.
func decodeArgOp(argop int) (int, int, int, int) {
	if argop < 0 || argop >= numArgOps {
		log.Fatalf("DecodeArgOp: invalid op range")
	}

	value := argop % numClasses
	argop = argop / numClasses

	key := argop % numClasses
	argop = argop / numClasses

	addr := argop % numClasses
	argop = argop / numClasses

	op := argop

	if !checkArgOp(op, addr, key, value) {
		log.Fatalf("DecodeArgOp: invalid operation/arguments")
	}
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
		contract, key, value = noArgID, noArgID, noArgID
	case 1:
		if !validateArg(opc[2]) {
			log.Fatalf("DecodeOpcode: wrong argument code")
		}
		contract, key, value = argId[opc[2]], noArgID, noArgID
	case 2:
		if !validateArg(opc[2]) || !validateArg(opc[3]) {
			log.Fatalf("DecodeOpcode: wrong argument code")
		}
		contract, key, value = argId[opc[2]], argId[opc[3]], noArgID
	case 3:
		if !validateArg(opc[2]) || !validateArg(opc[3]) || !validateArg(opc[4]) {
			log.Fatalf("DecodeOpcode: wrong argument code")
		}
		contract, key, value = argId[opc[2]], argId[opc[3]], argId[opc[4]]
	}
	return op, contract, key, value
}
