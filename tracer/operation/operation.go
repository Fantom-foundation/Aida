package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/substate-cli/state"
)

// Operation IDs of the StateDB interface
const (	GetStateID = iota
	SetStateID
	GetCommittedStateID
	SnapshotID
	RevertToSnapshotID
	CreateAccountID 
	GetBalanceID
	GetCodeHashID
	SuicideID
	ExistID
	FinaliseID 
	EndTransactionID
	BeginBlockID
	EndBlockID

	// Number of operations (must be last)
	NumOperations
)

// Dictionary data structure contains label and read function of an operation
type OperationDictionary struct {
	label string
	readfunc func(*os.File) (Operation, error)
}

// opDict contains a dictionary of operation's label and read function
var opDict = map[byte]OperationDictionary{
	GetStateID:		OperationDictionary{label: "GetState", readfunc: ReadGetState},
	SetStateID:		OperationDictionary{label: "SetState", readfunc: ReadSetState},
	GetCommittedStateID:	OperationDictionary{label: "GetCommittedState", readfunc: ReadGetCommittedState},
	SnapshotID:		OperationDictionary{label: "Snapshot", readfunc: ReadSnapshot},
	RevertToSnapshotID:	OperationDictionary{label: "RevertToSnapshot", readfunc: ReadRevertToSnapshot},
	CreateAccountID:	OperationDictionary{label: "CreateAccount", readfunc: ReadCreateAccount},
	GetBalanceID:		OperationDictionary{label: "GetBalance", readfunc: ReadGetBalance},
	GetCodeHashID:		OperationDictionary{label: "GetCodeHash", readfunc: ReadGetCodeHash},
	SuicideID:		OperationDictionary{label: "Suicide", readfunc: ReadSuicide},
	ExistID:		OperationDictionary{label: "Exist", readfunc: ReadExist},
	FinaliseID:		OperationDictionary{label: "Finalise", readfunc: ReadFinalise},
	EndTransactionID:	OperationDictionary{label: "EndTransaction", readfunc: ReadEndTransaction},
	BeginBlockID:		OperationDictionary{label: "BeginBlock", readfunc: ReadBeginBlock},
	EndBlockID:		OperationDictionary{label: "EndBlock", readfunc: ReadEndBlock},
}

// Get a label of a state operation
func getLabel(i byte) string {
	if i < 0 || i >= NumOperations {
		log.Fatalf("getLabel failed; index is out-of-bound")
	}
	return opDict[i].label
}

////////////////////////////////////////////////////////////
// State Operation Interface
////////////////////////////////////////////////////////////

// State-operation interface
type Operation interface {
	GetOpId() byte                             // obtain operation identifier
	writeOperation(*os.File)                   // write operation
	Execute(state.StateDB, *dict.DictionaryContext) // execute operation
	Debug(*dict.DictionaryContext)                  // print debug message for operation
}

// Read a state operation from file.
func ReadOperation(f *os.File) Operation {
	var (
		op Operation
		ID byte
	)

	// read ID from file
	err := binary.Read(f, binary.LittleEndian, &ID)
	if err == io.EOF {
		return nil
	} else if err != nil {
		log.Fatalf("Cannot read ID from file. Error:%v", err)
	}
	if ID >= NumOperations {
		log.Fatalf("ID out of range %v", ID)
	}

	// read state operation
	op, err = opDict[ID].readfunc(f)
	if err != nil {
		log.Fatalf("Failed to read operation %v. Error %v", getLabel(ID), err)
	}
	if op.GetOpId() != ID {
		log.Fatalf("Generated object of type %v has wrong ID (%v) ", getLabel(op.GetOpId()), getLabel(ID))
	}
	return op
}

// Write state operation to file.
func WriteOperation(f *os.File, op Operation) {
	// write ID to file
	ID := op.GetOpId()
	if err := binary.Write(f, binary.LittleEndian, &ID); err != nil {
		log.Fatalf("Failed to write ID for operation %v. Error: %v", getLabel(ID), err)
	}

	// write details of operation to file
	op.writeOperation(f)
}

// Write slice in little-endian format to file (helper Function).
func writeStruct(f *os.File, data any) {
	if err := binary.Write(f, binary.LittleEndian, data); err != nil {
		log.Fatalf("Failed to write binary data: %v", err)
	}
}

// Print debug information of a state operation.
func Debug(ctx *dict.DictionaryContext, op Operation) {
	fmt.Printf("%v:\n", getLabel(op.GetOpId()))
	op.Debug(ctx)
}
