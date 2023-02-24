package stochastic

import (
	"fmt"
	"log"
)

// IDs of StateDB Operations
const (
	AddBalanceID = iota
	BeginBlockID
	BeginEpochID
	BeginTransactionID
	CreateAccountID
	EmptyID
	EndBlockID
	EndEpochID
	EndTransactionID
	ExistID
	FinaliseID
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

// opText translates IDs to operation's text
var opText = map[int]string{
	AddBalanceID:        "AddBalance",
	BeginBlockID:        "BeginBlock",
	BeginEpochID:        "BeginEpoch",
	BeginTransactionID:  "BeginTransaction",
	CreateAccountID:     "CreateAccount",
	EmptyID:             "Empty",
	EndBlockID:          "EndBlock",
	EndEpochID:          "EndEpoch",
	EndTransactionID:    "EndTransaction",
	ExistID:             "Exist",
	FinaliseID:          "Finalise",
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
}

// opMnemo is a mnemonics table for operations.
var opMnemo = map[int]string{
	AddBalanceID:        "AB",
	BeginBlockID:        "BB",
	BeginEpochID:        "BE",
	BeginTransactionID:  "BT",
	CreateAccountID:     "CA",
	EmptyID:             "EM",
	EndBlockID:          "EB",
	EndEpochID:          "EE",
	EndTransactionID:    "ET",
	ExistID:             "EX",
	FinaliseID:          "FI",
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
}

// opNumArgs is an argument number table for operations.
var opNumArgs = map[int]int{
	AddBalanceID:        1,
	BeginBlockID:        0,
	BeginEpochID:        0,
	BeginTransactionID:  0,
	CreateAccountID:     1,
	EmptyID:             1,
	EndBlockID:          0,
	EndEpochID:          0,
	EndTransactionID:    0,
	ExistID:             1,
	FinaliseID:          0,
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
}

// opId is an operation ID table.
var opId = map[string]int{
	"AB": AddBalanceID,
	"BB": BeginBlockID,
	"BE": BeginEpochID,
	"BT": BeginTransactionID,
	"CA": CreateAccountID,
	"EM": EmptyID,
	"EB": EndBlockID,
	"EE": EndEpochID,
	"ET": EndTransactionID,
	"EX": ExistID,
	"FI": FinaliseID,
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
