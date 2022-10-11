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
// Stored Operations
const GetStateID = 0
const SetStateID = 1
const GetCommittedStateID = 2
const SnapshotID = 3
const RevertToSnapshotID = 4
const CreateAccountID = 5
const GetBalanceID = 6
const GetCodeHashID = 7
const SuicideID = 8
const ExistID = 9
const FinaliseID = 10
const EndTransactionID = 11 //last
const BeginBlockID = 12
const EndBlockID = 13

// Number of state operation identifiers
const NumOperations = EndBlockID + 1 //last op + 1

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

////////////////////////////////////////////////////////////
// Begin Block Operation (Pseudo Operation)
////////////////////////////////////////////////////////////

// Begin-block operation data structure
type BeginBlock struct {
	BlockNumber uint64 // block number
}

// Return the begin-block operation identifier.
func (op *BeginBlock) GetOpId() byte {
	return BeginBlockID
}

// Create a new begin-block operation.
func NewBeginBlock(bbNum uint64) *BeginBlock {
	return &BeginBlock{BlockNumber: bbNum}
}

// Read a begin-block operation from file.
func ReadBeginBlock(file *os.File) (Operation, error) {
	data := new(BeginBlock)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the begin-block operation to file.
func (op *BeginBlock) writeOperation(f *os.File) {
	if err := binary.Write(f, binary.LittleEndian, *op); err != nil {
		log.Fatalf("Failed to write binary data: %v", err)
	}
}

// Execute the begin-block operation.
func (op *BeginBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
}

// Print a debug message for begin-block.
func (op *BeginBlock) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tblock number: %v\n", op.BlockNumber)
}

////////////////////////////////////////////////////////////
// End Block Operation (Pseudo Operation)
////////////////////////////////////////////////////////////

// End-block operation data structure
type EndBlock struct {
}

// Return the end-block operation identifier.
func (op *EndBlock) GetOpId() byte {
	return EndBlockID
}

// Create a new end-block operation.
func NewEndBlock() *EndBlock {
	return &EndBlock{}
}

// Read an end-block operation from file.
func ReadEndBlock(file *os.File) (Operation, error) {
	return NewEndBlock(), nil
}

// Write the end-block operation to file.
func (op *EndBlock) writeOperation(f *os.File) {
}

// Execute the end-block operation.
func (op *EndBlock) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
}

// Print a debug message for end-block.
func (op *EndBlock) Debug(ctx *dict.DictionaryContext) {
}

////////////////////////////////////////////////////////////
// GetState Operation
////////////////////////////////////////////////////////////

// Get-state data structure
type GetState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
}

// Return the get-state operation identifier.
func (op *GetState) GetOpId() byte {
	return GetStateID
}

// Create a new get-state operation.
func NewGetState(cIdx uint32, sIdx uint32) *GetState {
	return &GetState{ContractIndex: cIdx, StorageIndex: sIdx}
}

// Read a get-state operation from a file.
func ReadGetState(file *os.File) (Operation, error) {
	data := new(GetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-state operation to file.
func (op *GetState) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex, op.StorageIndex}
	writeStruct(f, op)
}

// Execute the get-state operation.
func (op *GetState) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	db.GetState(contract, storage)
}

// Print a debug message.
func (op *GetState) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\n",
		ctx.DecodeContract(op.ContractIndex),
		ctx.DecodeStorage(op.StorageIndex))
}

////////////////////////////////////////////////////////////
// SetState Operation
////////////////////////////////////////////////////////////

// Set-state data structure
type SetState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
	ValueIndex    uint64 // encoded storage value
}

// Return the set-state identifier
func (op *SetState) GetOpId() byte {
	return SetStateID
}

// Create a new set-state operation.
func NewSetState(cIdx uint32, sIdx uint32, vIdx uint64) *SetState {
	return &SetState{ContractIndex: cIdx, StorageIndex: sIdx, ValueIndex: vIdx}
}

// Read a set-state operation from file.
func ReadSetState(file *os.File) (Operation, error) {
	data := new(SetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the set-state operation to file.
func (op *SetState) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex, op.StorageIndex, op.ValueIndex}
	writeStruct(f, op)
}

// Execute the set-state operation.
func (op *SetState) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	value := ctx.DecodeValue(op.ValueIndex)
	db.SetState(contract, storage, value)
}

// Print a debug message for set-state.
func (op *SetState) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\t value: %v\n",
		ctx.DecodeContract(op.ContractIndex),
		ctx.DecodeStorage(op.StorageIndex),
		ctx.DecodeValue(op.ValueIndex))
}

////////////////////////////////////////////////////////////
// GetCommittedState Operation
////////////////////////////////////////////////////////////

// Get-committed-state data structure
type GetCommittedState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
}

// Return the get-commited-state-operation identifier.
func (op *GetCommittedState) GetOpId() byte {
	return GetCommittedStateID
}

// Create a new get-commited-state operation.
func NewGetCommittedState(cIdx uint32, sIdx uint32) *GetCommittedState {
	return &GetCommittedState{ContractIndex: cIdx, StorageIndex: sIdx}
}

// Read a get-commited-state operation from file.
func ReadGetCommittedState(file *os.File) (Operation, error) {
	data := new(GetCommittedState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-commited-state operation to file.
func (op *GetCommittedState) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex, op.StorageIndex}
	writeStruct(f, op)
}

// Execute the get-committed-state operation.
func (op *GetCommittedState) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	storage := ctx.DecodeStorage(op.StorageIndex)
	db.GetCommittedState(contract, storage)
}

// Print debug message for get-committed-state.
func (op *GetCommittedState) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\n",
		ctx.DecodeContract(op.ContractIndex),
		ctx.DecodeStorage(op.StorageIndex))
}

////////////////////////////////////////////////////////////
// Snapshot Operation
////////////////////////////////////////////////////////////

// Snapshot data structure
type Snapshot struct {
}

// Return the snapshot operation identifier.
func (op *Snapshot) GetOpId() byte {
	return SnapshotID
}

// Create a new snapshot operation.
func NewSnapshot() *Snapshot {
	return &Snapshot{}
}

// Read a snapshot operation from a file.
func ReadSnapshot(file *os.File) (Operation, error) {
	return NewSnapshot(), nil
}

// Write the snapshot operation to file.
func (op *Snapshot) writeOperation(f *os.File) {
}

// Execute the snapshot operation.
func (op *Snapshot) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	db.Snapshot()
}

// Print the details for the snapshot operation.
func (op *Snapshot) Debug(*dict.DictionaryContext) {
}

////////////////////////////////////////////////////////////
// RevertToSnapshot Operation
////////////////////////////////////////////////////////////

// Revert-to-snapshot operation's data structure with returned snapshot id
type RevertToSnapshot struct {
	SnapshotID int
}

// Return the revert-to-snapshot operation identifier.
func (op *RevertToSnapshot) GetOpId() byte {
	return RevertToSnapshotID
}

// Create a new revert-to-snapshot operation.
func NewRevertToSnapshot(SnapshotID int) *RevertToSnapshot {
	return &RevertToSnapshot{SnapshotID: SnapshotID}
}

// Read a revert-to-snapshot operation from file.
func ReadRevertToSnapshot(file *os.File) (Operation, error) {
	var data int32
	err := binary.Read(file, binary.LittleEndian, &data)
	op := &RevertToSnapshot{SnapshotID: int(data)}
	return op, err
}

// Write the revert-to-snapshot operation to file.
func (op *RevertToSnapshot) writeOperation(f *os.File) {
	data := int32(op.SnapshotID)
	writeStruct(f, data)
}

// Execute the revert-to-snapshot operation.
func (op *RevertToSnapshot) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	db.RevertToSnapshot(op.SnapshotID)
}

// Print a debug message for revert-to-snapshot operation.
func (op *RevertToSnapshot) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tsnapshot id: %v\n", op.SnapshotID)
}

////////////////////////////////////////////////////////////
// CreateAccount Operation
////////////////////////////////////////////////////////////

// Create-account data structure
type CreateAccount struct {
	ContractIndex uint32 // encoded contract address
}

// Return the create-account operation identifier.
func (op *CreateAccount) GetOpId() byte {
	return CreateAccountID
}

// Create a new create account operation.
func NewCreateAccount(cIdx uint32) *CreateAccount {
	return &CreateAccount{ContractIndex: cIdx}
}

// Read a create-account operation from a file.
func ReadCreateAccount(file *os.File) (Operation, error) {
	data := new(CreateAccount)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the create account operation to file.
func (op *CreateAccount) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex}
	writeStruct(f, op)
}

// Execute the create account operation.
func (op *CreateAccount) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.CreateAccount(contract)
}

// Print a debug message for snapshot operation.
func (op *CreateAccount) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// GetBalance Operation
////////////////////////////////////////////////////////////

// GetBalance data structure
type GetBalance struct {
	ContractIndex uint32
}

// Return the get-balance operation identifier.
func (op *GetBalance) GetOpId() byte {
	return GetBalanceID
}

// Create a new get-balance operation.
func NewGetBalance(cIdx uint32) *GetBalance {
	return &GetBalance{ContractIndex: cIdx}
}

// Read a get-balance operation from a file.
func ReadGetBalance(file *os.File) (Operation, error) {
	data := new(GetBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-balance operation.
func (op *GetBalance) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex}
	writeStruct(f, op)
}

// Execute the get-balance operation.
func (op *GetBalance) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.GetBalance(contract)
}

// Print a debug message for get-balance.
func (op *GetBalance) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// GetCodeHash Operation
////////////////////////////////////////////////////////////

// Get-code-hash data structure
type GetCodeHash struct {
	ContractIndex uint32 // encoded contract address
}

// Return the get-code-hash operation identifier.
func (op *GetCodeHash) GetOpId() byte {
	return GetCodeHashID
}

// Create a new get-code-hash operation.
func NewGetCodeHash(cIdx uint32) *GetCodeHash {
	return &GetCodeHash{ContractIndex: cIdx}
}

// Read a get-code-hash operation from a file.
func ReadGetCodeHash(file *os.File) (Operation, error) {
	data := new(GetCodeHash)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the get-code-hash operation to a file.
func (op *GetCodeHash) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex}
	writeStruct(f, op)
}

// Execute the get-code-hash operation.
func (op *GetCodeHash) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.GetCodeHash(contract)
}

// Print a debug message for get-code-hash.
func (op *GetCodeHash) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// Suicide Operation
////////////////////////////////////////////////////////////

// Suicide data structure
type Suicide struct {
	ContractIndex uint32 // encoded contract address
}

// Return the suicide operation identifier.
func (op *Suicide) GetOpId() byte {
	return SuicideID
}

// Create a new suicide operation.
func NewSuicide(cIdx uint32) *Suicide {
	return &Suicide{ContractIndex: cIdx}
}

// Read a suicide operation from a file.
func ReadSuicide(file *os.File) (Operation, error) {
	data := new(Suicide)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the suicide operation to a file.
func (op *Suicide) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex}
	writeStruct(f, op)
}

// Execute the suicide operation.
func (op *Suicide) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.Suicide(contract)
}

// Print a debug message for suicide.
func (op *Suicide) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// Exist Operation
////////////////////////////////////////////////////////////

// Exist data structure
type Exist struct {
	ContractIndex uint32 // encoded contract address
}

// Return the exist operation identifier.
func (op *Exist) GetOpId() byte {
	return ExistID
}

// Create a new exist operation.
func NewExist(cIdx uint32) *Exist {
	return &Exist{ContractIndex: cIdx}
}

// Read a exist operation from a file.
func ReadExist(file *os.File) (Operation, error) {
	data := new(Exist)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the exist operation to a file.
func (op *Exist) writeOperation(f *os.File) {
	//var data = []any{op.ContractIndex}
	writeStruct(f, op)
}

// Execute the exist operation.
func (op *Exist) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	contract := ctx.DecodeContract(op.ContractIndex)
	db.Exist(contract)
}

// Print a debug message for exist.
func (op *Exist) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.DecodeContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// Finalise Operation
////////////////////////////////////////////////////////////

// Finalise data structure
type Finalise struct {
	DeleteEmptyObjects bool
}

// Return the finalise operation identifier.
func (op *Finalise) GetOpId() byte {
	return FinaliseID
}

// Create a new finalise operation.
func NewFinalise(deleteEmptyObjects bool) *Finalise {
	return &Finalise{DeleteEmptyObjects: deleteEmptyObjects}
}

// Read a finalise operation from a file.
func ReadFinalise(file *os.File) (Operation, error) {
	data := new(Finalise)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write the finalise operation to a file.
func (op *Finalise) writeOperation(f *os.File) {
	//var data = []any{op.DeleteEmptyObjects}
	writeStruct(f, op)
}

// Execute the finalise operation.
func (op *Finalise) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
	db.Finalise(op.DeleteEmptyObjects)
}

// Print a debug message for finalise.
func (op *Finalise) Debug(ctx *dict.DictionaryContext) {
	fmt.Printf("\tdelete empty objects: %v\n", op.DeleteEmptyObjects)
}

////////////////////////////////////////////////////////////
// End of transaction Operation
////////////////////////////////////////////////////////////

// End-transaction operation's data structure
type EndTransaction struct {
}

// Return the end-transaction operation identifier.
func (op *EndTransaction) GetOpId() byte {
	return EndTransactionID
}

// Create a new end-transaction operation.
func NewEndTransaction() *EndTransaction {
	return &EndTransaction{}
}

// Read a new end-transaction operation from file.
func ReadEndTransaction(*os.File) (Operation, error) {
	return new(EndTransaction), nil
}

// Write the end-transaction operation to file.
func (op *EndTransaction) writeOperation(f *os.File) {
}

// Execute the end-transaction operation.
func (op *EndTransaction) Execute(db state.StateDB, ctx *dict.DictionaryContext) {
}

// Print a debug message for end-transaction.
func (op *EndTransaction) Debug(*dict.DictionaryContext) {
}
