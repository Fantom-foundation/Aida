package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/Fantom-foundation/Aida/tracer/state"
)

var stats *ProfileStats = new(ProfileStats)

// Operation IDs of the StateDB interface
const (
	AddBalanceID = iota
	BeginBlockID
	CreateAccountID
	EmptyID
	EndBlockID
	EndTransactionID
	ExistID
	FinaliseID
	GetBalanceID
	GetCodeHashID
	GetCodeHashLcID
	GetCodeID
	GetCodeSizeID
	GetCommittedStateID
	GetCommittedStateLclsID
	GetNonceID
	GetStateID
	GetStateLcID
	GetStateLccsID
	GetStateLclsID
	HasSuicidedID
	RevertToSnapshotID
	SetCodeID
	SetNonceID
	SetStateID
	SetStateLclsID
	SnapshotID
	SubBalanceID
	SuicideID

	AddAddressToAccessListID
	AddLogID
	AddPreimageID
	AddRefundID
	AddressInAccessListID
	AddSlotToAccessListID
	CloseID
	ForEachStorageID
	GetLogsID
	GetRefundID
	IntermediateRootID
	PrepareAccessListID
	PrepareID
	SlotInAccessListID
	SubRefundID

	// NumProfiledOperations is number of profiled operations (must be last)
	NumProfiledOperations
)

// OperationDictionary data structure contains a label and a read function for an operation
type OperationDictionary struct {
	label    string                             // operation's label
	readfunc func(io.Reader) (Operation, error) // operation's read-function
}

// opDict relates an operation's id with its label and read-function.
var opDict = map[byte]OperationDictionary{
	AddBalanceID:            {label: "AddBalance", readfunc: ReadAddBalance},
	BeginBlockID:            {label: "BeginBlock", readfunc: ReadBeginBlock},
	CreateAccountID:         {label: "CreateAccount", readfunc: ReadCreateAccount},
	EmptyID:                 {label: "Exist", readfunc: ReadEmpty},
	EndBlockID:              {label: "EndBlock", readfunc: ReadEndBlock},
	EndTransactionID:        {label: "EndTransaction", readfunc: ReadEndTransaction},
	ExistID:                 {label: "Exist", readfunc: ReadExist},
	FinaliseID:              {label: "Finalise", readfunc: ReadFinalise},
	GetBalanceID:            {label: "GetBalance", readfunc: ReadGetBalance},
	GetCodeHashID:           {label: "GetCodeHash", readfunc: ReadGetCodeHash},
	GetCodeHashLcID:         {label: "GetCodeLcHash", readfunc: ReadGetCodeHashLc},
	GetCodeID:               {label: "GetCode", readfunc: ReadGetCode},
	GetCodeSizeID:           {label: "GetCodeSize", readfunc: ReadGetCodeSize},
	GetCommittedStateID:     {label: "GetCommittedState", readfunc: ReadGetCommittedState},
	GetCommittedStateLclsID: {label: "GetCommittedStateLcls", readfunc: ReadGetCommittedStateLcls},
	GetNonceID:              {label: "GetNonce", readfunc: ReadGetNonce},
	GetStateID:              {label: "GetState", readfunc: ReadGetState},
	GetStateLcID:            {label: "GetStateLc", readfunc: ReadGetStateLc},
	GetStateLccsID:          {label: "GetStateLccs", readfunc: ReadGetStateLccs},
	GetStateLclsID:          {label: "GetStateLcls", readfunc: ReadGetStateLcls},
	HasSuicidedID:           {label: "HasSuicided", readfunc: ReadHasSuicided},
	RevertToSnapshotID:      {label: "RevertToSnapshot", readfunc: ReadRevertToSnapshot},
	SetCodeID:               {label: "SetCode", readfunc: ReadSetCode},
	SetNonceID:              {label: "SetNonce", readfunc: ReadSetNonce},
	SetStateID:              {label: "SetState", readfunc: ReadSetState},
	SetStateLclsID:          {label: "SetStateLcls", readfunc: ReadSetStateLcls},
	SnapshotID:              {label: "Snapshot", readfunc: ReadSnapshot},
	SubBalanceID:            {label: "SubBalance", readfunc: ReadSubBalance},
	SuicideID:               {label: "Suicide", readfunc: ReadSuicide},
}

// GetLabel retrieves a label of a state operation.
func GetLabel(i byte) string {
	if i < 0 || i >= NumProfiledOperations {
		log.Fatalf("GetLabel failed; index is out-of-bound")
	}
	if _, ok := opDict[i]; !ok {
		log.Fatalf("operation is not defined")
	}

	return opDict[i].label
}

// Operation interface.
type Operation interface {
	GetId() byte                                                  // get operation identifier
	Write(io.Writer) error                                        // write operation to a file
	Execute(state.StateDB, *dict.DictionaryContext) time.Duration // execute operation on a stateDB instance
	Debug(*dict.DictionaryContext)                                // print debug message for operation
}

// Read an operation from file.
func Read(f io.Reader) Operation {
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
	if ID >= NumProfiledOperations {
		log.Fatalf("ID out of range %v", ID)
	}

	// read state operation
	op, err = opDict[ID].readfunc(f)
	if err != nil {
		log.Fatalf("Failed to read operation %v. Error %v", GetLabel(ID), err)
	}
	if op.GetId() != ID {
		log.Fatalf("Generated object of type %v has wrong ID (%v) ", GetLabel(op.GetId()), GetLabel(ID))
	}
	return op
}

// Write an operation to file.
func Write(f io.Writer, op Operation) {
	// write ID to file
	ID := op.GetId()
	if err := binary.Write(f, binary.LittleEndian, &ID); err != nil {
		log.Fatalf("Failed to write ID for operation %v. Error: %v", GetLabel(ID), err)
	}

	// write details of operation to file
	if err := op.Write(f); err != nil {
		log.Fatalf("Failed to write operation %v. Error: %v", GetLabel(ID), err)
	}
}

// Execute an operation and profile it.
func Execute(op Operation, db state.StateDB, ctx *dict.DictionaryContext) {
	elapsed := op.Execute(db, ctx)
	if EnableProfiling {
		stats.Profile(op.GetId(), elapsed)
	}
}

func PrintProfiling() {
	stats.PrintProfiling()
}

// Debug prints debug information of an operation.
func Debug(ctx *dict.DictionaryContext, op Operation) {
	fmt.Printf("\t%s: ", GetLabel(op.GetId()))
	op.Debug(ctx)
	fmt.Println()
}
