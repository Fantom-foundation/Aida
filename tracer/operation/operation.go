package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Fantom-foundation/aida/tracer/dict"
	"github.com/Fantom-foundation/aida/tracer/state"
)

// Operation IDs of the StateDB interface
const (
	GetStateID = iota
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
	label    string
	readfunc func(*os.File) (Operation, error)
}

// opDict contains a dictionary of operation's label and read function.
var opDict = map[byte]OperationDictionary{
	GetStateID:          {label: "GetState", readfunc: ReadGetState},
	SetStateID:          {label: "SetState", readfunc: ReadSetState},
	GetCommittedStateID: {label: "GetCommittedState", readfunc: ReadGetCommittedState},
	SnapshotID:          {label: "Snapshot", readfunc: ReadSnapshot},
	RevertToSnapshotID:  {label: "RevertToSnapshot", readfunc: ReadRevertToSnapshot},
	CreateAccountID:     {label: "CreateAccount", readfunc: ReadCreateAccount},
	GetBalanceID:        {label: "GetBalance", readfunc: ReadGetBalance},
	GetCodeHashID:       {label: "GetCodeHash", readfunc: ReadGetCodeHash},
	SuicideID:           {label: "Suicide", readfunc: ReadSuicide},
	ExistID:             {label: "Exist", readfunc: ReadExist},
	FinaliseID:          {label: "Finalise", readfunc: ReadFinalise},
	EndTransactionID:    {label: "EndTransaction", readfunc: ReadEndTransaction},
	BeginBlockID:        {label: "BeginBlock", readfunc: ReadBeginBlock},
	EndBlockID:          {label: "EndBlock", readfunc: ReadEndBlock},
}

// Get a label of a state operation.
func getLabel(i byte) string {
	if i < 0 || i >= NumOperations {
		log.Fatalf("getLabel failed; index is out-of-bound")
	}
	return opDict[i].label
}

// Operation interface.
type Operation interface {
	GetOpId() byte                                  // obtain operation identifier
	Write(*os.File) error                           // write operation
	Execute(state.StateDB, *dict.DictionaryContext) // execute operation
	Debug(*dict.DictionaryContext)                  // print debug message for operation
}

// Read an operation from file.
func Read(f *os.File) Operation {
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

// Write an operation to file.
func Write(f *os.File, op Operation) {
	// write ID to file
	ID := op.GetOpId()
	if err := binary.Write(f, binary.LittleEndian, &ID); err != nil {
		log.Fatalf("Failed to write ID for operation %v. Error: %v", getLabel(ID), err)
	}

	// write details of operation to file
	if err := op.Write(f); err != nil {
		log.Fatalf("Failed to write operation %v. Error: %v", getLabel(ID), err)
	}
}

// Print debug information of an operation.
func Debug(ctx *dict.DictionaryContext, op Operation) {
	fmt.Printf("%v:\n", getLabel(op.GetOpId()))
	op.Debug(ctx)
}
