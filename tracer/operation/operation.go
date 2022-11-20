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

	// NumProfiledOperations is number of profiled operations (must be last)
	NumProfiledOperations

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
)

// OperationDictionary data structure contains a Label and a read function for an operation
type OperationDictionary struct {
	Label    string                             // operation's Label
	Readfunc func(io.Reader) (Operation, error) // operation's read-function
}

// OpDict relates an operation's id with its Label and read-function.
var OpDict = map[byte]OperationDictionary{
	AddBalanceID:            {Label: "AddBalance", Readfunc: ReadAddBalance},
	BeginBlockID:            {Label: "BeginBlock", Readfunc: ReadBeginBlock},
	CreateAccountID:         {Label: "CreateAccount", Readfunc: ReadCreateAccount},
	EmptyID:                 {Label: "Exist", Readfunc: ReadEmpty},
	EndBlockID:              {Label: "EndBlock", Readfunc: ReadEndBlock},
	EndTransactionID:        {Label: "EndTransaction", Readfunc: ReadEndTransaction},
	ExistID:                 {Label: "Exist", Readfunc: ReadExist},
	FinaliseID:              {Label: "Finalise", Readfunc: ReadFinalise},
	GetBalanceID:            {Label: "GetBalance", Readfunc: ReadGetBalance},
	GetCodeHashID:           {Label: "GetCodeHash", Readfunc: ReadGetCodeHash},
	GetCodeHashLcID:         {Label: "GetCodeLcHash", Readfunc: ReadGetCodeHashLc},
	GetCodeID:               {Label: "GetCode", Readfunc: ReadGetCode},
	GetCodeSizeID:           {Label: "GetCodeSize", Readfunc: ReadGetCodeSize},
	GetCommittedStateID:     {Label: "GetCommittedState", Readfunc: ReadGetCommittedState},
	GetCommittedStateLclsID: {Label: "GetCommittedStateLcls", Readfunc: ReadGetCommittedStateLcls},
	GetNonceID:              {Label: "GetNonce", Readfunc: ReadGetNonce},
	GetStateID:              {Label: "GetState", Readfunc: ReadGetState},
	GetStateLcID:            {Label: "GetStateLc", Readfunc: ReadGetStateLc},
	GetStateLccsID:          {Label: "GetStateLccs", Readfunc: ReadGetStateLccs},
	GetStateLclsID:          {Label: "GetStateLcls", Readfunc: ReadGetStateLcls},
	HasSuicidedID:           {Label: "HasSuicided", Readfunc: ReadHasSuicided},
	RevertToSnapshotID:      {Label: "RevertToSnapshot", Readfunc: ReadRevertToSnapshot},
	SetCodeID:               {Label: "SetCode", Readfunc: ReadSetCode},
	SetNonceID:              {Label: "SetNonce", Readfunc: ReadSetNonce},
	SetStateID:              {Label: "SetState", Readfunc: ReadSetState},
	SetStateLclsID:          {Label: "SetStateLcls", Readfunc: ReadSetStateLcls},
	SnapshotID:              {Label: "Snapshot", Readfunc: ReadSnapshot},
	SubBalanceID:            {Label: "SubBalance", Readfunc: ReadSubBalance},
	SuicideID:               {Label: "Suicide", Readfunc: ReadSuicide},

	// for testing
	AddAddressToAccessListID: {Label: "AddAddressToAccessList", Readfunc: ReadPanic},
	AddLogID:                 {Label: "AddLog", Readfunc: ReadPanic},
	AddPreimageID:            {Label: "AddPreimage", Readfunc: ReadPanic},
	AddRefundID:              {Label: "AddRefund", Readfunc: ReadPanic},
	AddressInAccessListID:    {Label: "AddressInAccessList", Readfunc: ReadPanic},
	AddSlotToAccessListID:    {Label: "AddSlotToAccessList", Readfunc: ReadPanic},
	CloseID:                  {Label: "Close", Readfunc: ReadPanic},
	ForEachStorageID:         {Label: "ForEachStorage", Readfunc: ReadPanic},
	GetLogsID:                {Label: "GetLogs", Readfunc: ReadPanic},
	GetRefundID:              {Label: "GetRefund", Readfunc: ReadPanic},
	IntermediateRootID:       {Label: "IntermediateRoot", Readfunc: ReadPanic},
	PrepareAccessListID:      {Label: "PrepareAccessList", Readfunc: ReadPanic},
	PrepareID:                {Label: "Prepare", Readfunc: ReadPanic},
	SlotInAccessListID:       {Label: "SlotInAccessList", Readfunc: ReadPanic},
	SubRefundID:              {Label: "SubRefund", Readfunc: ReadPanic},
}

// GetLabel retrieves a Label of a state operation.
func GetLabel(i byte) string {
	if _, ok := OpDict[i]; !ok {
		log.Fatalf("GetLabel failed; operation is not defined")
	}

	return OpDict[i].Label
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
	op, err = OpDict[ID].Readfunc(f)
	if err != nil {
		log.Fatalf("Failed to read operation %v. Error %v", GetLabel(ID), err)
	}
	if op.GetId() != ID {
		log.Fatalf("Generated object of type %v has wrong ID (%v) ", GetLabel(op.GetId()), GetLabel(ID))
	}
	return op
}

func ReadPanic(file io.Reader) (Operation, error) {
	panic("operation not implemented")
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
