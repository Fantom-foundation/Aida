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

package operation

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
)

// Operation IDs of the StateDB interface
const (
	AddBalanceID = iota
	BeginBlockID
	BeginSyncPeriodID
	BeginTransactionID
	CreateAccountID
	CommitID
	EmptyID
	EndBlockID
	EndSyncPeriodID
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
	GetStateLccsID
	GetStateLcID
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
	AddressInAccessListID
	AddSlotToAccessListID
	PrepareAccessListID
	SlotInAccessListID

	AddLogID
	AddPreimageID
	AddRefundID
	CloseID
	ForEachStorageID
	GetLogsID
	GetRefundID
	IntermediateRootID
	PrepareID
	SubRefundID

	// WARNING: New IDs should be added here. Any change in the order of the
	// IDs above invalidates persisted data -- in particular storage traces.

	// NumOperations is number of distinct operations (must be last)
	NumOperations
)

// OperationDictionary data structure contains a Label and a read function for an operation
type OperationDictionary struct {
	label    string                             // operation's Label
	readfunc func(io.Reader) (Operation, error) // operation's read-function
}

// opDict relates an operation's id with its label and read-function.
var opDict = map[byte]OperationDictionary{
	AddBalanceID:            {label: "AddBalance", readfunc: ReadAddBalance},
	BeginBlockID:            {label: "BeginBlock", readfunc: ReadBeginBlock},
	BeginSyncPeriodID:       {label: "BeginSyncPeriod", readfunc: ReadBeginSyncPeriod},
	BeginTransactionID:      {label: "BeginTransaction", readfunc: ReadBeginTransaction},
	CommitID:                {label: "Commit", readfunc: ReadPanic},
	CreateAccountID:         {label: "CreateAccount", readfunc: ReadCreateAccount},
	EmptyID:                 {label: "Empty", readfunc: ReadEmpty},
	EndBlockID:              {label: "EndBlock", readfunc: ReadEndBlock},
	EndSyncPeriodID:         {label: "EndSyncPeriod", readfunc: ReadEndSyncPeriod},
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

	// for testing
	AddAddressToAccessListID: {label: "AddAddressToAccessList", readfunc: ReadPanic},
	AddLogID:                 {label: "AddLog", readfunc: ReadPanic},
	AddPreimageID:            {label: "AddPreimage", readfunc: ReadPanic},
	AddRefundID:              {label: "AddRefund", readfunc: ReadPanic},
	AddressInAccessListID:    {label: "AddressInAccessList", readfunc: ReadPanic},
	AddSlotToAccessListID:    {label: "AddSlotToAccessList", readfunc: ReadPanic},
	CloseID:                  {label: "Close", readfunc: ReadPanic},
	ForEachStorageID:         {label: "ForEachStorage", readfunc: ReadPanic},
	GetLogsID:                {label: "GetLogs", readfunc: ReadPanic},
	GetRefundID:              {label: "GetRefund", readfunc: ReadPanic},
	IntermediateRootID:       {label: "IntermediateRoot", readfunc: ReadPanic},
	PrepareAccessListID:      {label: "PrepareAccessList", readfunc: ReadPanic},
	PrepareID:                {label: "Prepare", readfunc: ReadPanic},
	SlotInAccessListID:       {label: "SlotInAccessList", readfunc: ReadPanic},
	SubRefundID:              {label: "SubRefund", readfunc: ReadPanic},
}

// GetLabel retrieves a label of a state operation.
func GetLabel(i byte) string {
	if _, ok := opDict[i]; !ok {
		log.Fatalf("GetLabel failed; operation is not defined")
	}

	return opDict[i].label
}

// Operation interface.
type Operation interface {
	GetId() byte                                          // get operation identifier
	Write(io.Writer) error                                // write operation to a file
	Execute(state.StateDB, *context.Replay) time.Duration // execute operation on a stateDB instance
	Debug(*context.Context)                               // print debug message for operation
}

// Read an operation from file.
func Read(f io.Reader) (Operation, error) {
	var (
		op Operation
		ID byte
	)

	// read ID from file
	err := binary.Read(f, binary.LittleEndian, &ID)
	if err == io.EOF {
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("cannot read ID from file; %v", err)
	}
	if ID >= NumOperations {
		return nil, fmt.Errorf("operaiton ID out of range %v", ID)
	}

	// read state operation
	op, err = opDict[ID].readfunc(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read operation %v; %v", GetLabel(ID), err)
	}
	if op.GetId() != ID {
		return nil, fmt.Errorf("generated object of type %v has wrong ID (%v)", GetLabel(op.GetId()), GetLabel(ID))
	}
	return op, err
}

func ReadPanic(f io.Reader) (Operation, error) {
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
func Execute(op Operation, db state.StateDB, ctx *context.Replay) {
	elapsed := op.Execute(db, ctx)
	if ctx.Profile {
		ctx.Stats.Profile(op.GetId(), elapsed)
	}
}

// Debug prints debug information of an operation.
func Debug(ctx *context.Context, op Operation) {
	fmt.Printf("\t%s: ", GetLabel(op.GetId()))
	op.Debug(ctx)
	fmt.Println()
}

// writeOperation writes operation to file.
func WriteOp(ctx *context.Record, op Operation) {
	Write(ctx.ZFile, op)
	if ctx.Debug {
		Debug(&ctx.Context, op)
	}
}

// CreateIdLabelMap returns a map of opcode ID and opcode name
func CreateIdLabelMap() map[byte]string {
	ret := make(map[byte]string)
	for id := byte(0); id < NumOperations; id++ {
		ret[id] = GetLabel(id)
	}
	return ret
}
