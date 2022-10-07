package tracer

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

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

// State operations' names
var idToLabel = [NumOperations]string{
	"GetState",
	"SetState",
	"GetCommittedState",
	"Snapshot",
	"RevertToSnapshot",
	"CreateAccount",
	"GetBalance",
	"GetCodeHash",
	"Suicide",
	"Exist",
	"Finalise",
	"EndTransaction",
	// Pseudo Operations
	"BeginBlock",
	"EndBlock",
}

// State operation's read functions
var readFunction = [NumOperations]func(*os.File) (Operation, error){
	ReadGetState,
	ReadSetState,
	ReadGetCommittedState,
	ReadSnapshot,
	ReadRevertToSnapshot,
	ReadCreateAccount,
	ReadGetBalance,
	ReadGetCodeHash,
	ReadSuicide,
	ReadExist,
	ReadFinalise,
	ReadEndTransaction,
}

// Get a label of a state operation
func GetLabel(i byte) string {
	if i < 0 || i >= NumOperations {
		log.Fatalf("GetLabel failed; index is out-of-bound")
	}
	return idToLabel[i]
}

////////////////////////////////////////////////////////////
// State Operation Interface
////////////////////////////////////////////////////////////

// State-operation interface
type Operation interface {
	GetOpId() byte                             // obtain operation identifier
	WriteOperation(*os.File)                            // write operation
	Execute(state.StateDB, *DictionaryContext) // execute operation
	Debug(*DictionaryContext)                  // print debug message for operation
}

// Read a state operation from file.
// TODO: Rename Read to ReadOperation
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

	// read state operation in binary format from file
	op, err = readFunction[ID](f)
	if err != nil {
		log.Fatalf("Failed to read operation %v. Error %v", GetLabel(ID), err)
	}
	if op.GetOpId() != ID {
		log.Fatalf("Generated object of type %v has wrong ID (%v) ", GetLabel(op.GetOpId()), GetLabel(ID))
	}
	return op
}

// Write state operation to file.
// TODO: Rename Write to WriteOperation
func WriteOperation(f *os.File, op Operation) {
	// write ID to file
	ID := op.GetOpId()
	if err := binary.Write(f, binary.LittleEndian, &ID); err != nil {
		log.Fatalf("Failed to write ID for operation %v. Error: %v", GetLabel(ID), err)
	}

	// write details of operation to file
	op.WriteOperation(f)
}

// Write slice in little-endian format to file (helper Function).
func writeSlice(f *os.File, data []any) {
	for _, val := range data {
		if err := binary.Write(f, binary.LittleEndian, val); err != nil {
			log.Fatalf("Failed to write binary data: %v", err)
		}
	}
}

// Print debug information of a state operation.
func Debug(ctx *DictionaryContext, op Operation) {
	fmt.Printf("%v:\n", GetLabel(op.GetOpId()))
	op.Debug(ctx)
}

// TODO: Remove from Operation from following structs 

////////////////////////////////////////////////////////////
// Begin Block Operation (Pseudo Operation)
////////////////////////////////////////////////////////////

// Block-operation data structure capturing the beginning of a block.
type BeginBlock struct {
	BlockNumber uint64 // block number
}

// Return begin-block operation identifier.
func (bb *BeginBlock) GetOpId() byte {
	return BeginBlockID
}

// Create a new begin-block operation.
func NewBeginBlock(bbNum uint64) *BeginBlock {
	return &BeginBlock{BlockNumber: bbNum}
}

// Write block operation (should never be invoked).
func (bb *BeginBlock) WriteOperation(files *os.File) {
}

// Execute state operation.
func (bb *BeginBlock) Execute(db state.StateDB, ctx *DictionaryContext) {
}

// Print a debug message.
func (bb *BeginBlock) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tblock number: %v\n", bb.BlockNumber)
}

////////////////////////////////////////////////////////////
// End Block Operation (Pseudo Operation)
////////////////////////////////////////////////////////////

// Block-operation data structure capturing the beginning of a block.
type EndBlock struct {
	BlockNumber uint64 // block number
}

// Return end-block operation identifier.
func (eb *EndBlock) GetOpId() byte {
	return EndBlockID
}

// Create a new end-block operation.
func NewEndBlock(ebNum uint64) *EndBlock {
	return &EndBlock{BlockNumber: ebNum}
}

// Write end-block operation (should never be invoked).
func (eb *EndBlock) WriteOperation(files *os.File) {
}

// Execute state operation.
func (eb *EndBlock) Execute(db state.StateDB, ctx *DictionaryContext) {
}

// Print a debug message
func (eb *EndBlock) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tblock number: %v\n", eb.BlockNumber)
}

////////////////////////////////////////////////////////////
// GetState Operation
////////////////////////////////////////////////////////////

// GetState datastructure with encoded contract and storage addresses.
type GetState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
}

// Return get-state operation identifier.
func (op *GetState) GetOpId() byte {
	return GetStateID
}

// Create a new get-state operation.
func NewGetState(cIdx uint32, sIdx uint32) *GetState {
	return &GetState{ContractIndex: cIdx, StorageIndex: sIdx}
}

// Read get-state operation from a file.
func ReadGetState(file *os.File) (Operation, error) {
	data := new(GetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a get-state operation in binary format to a file.
func (op *GetState) WriteOperation(f *os.File) {
	var data = []any{op.ContractIndex, op.StorageIndex}
	writeSlice(f, data)
}

// Execute get-state operation.
func (op *GetState) Execute(db state.StateDB, ctx *DictionaryContext) {
	contract := ctx.getContract(op.ContractIndex)
	storage := ctx.getStorage(op.StorageIndex)
	db.GetState(contract, storage)
}

// Print a debug message.
func (op *GetState) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\n",
		ctx.getContract(op.ContractIndex),
		ctx.getStorage(op.StorageIndex))
}

////////////////////////////////////////////////////////////
// SetState Operation
////////////////////////////////////////////////////////////

// SetState datastructure with encoded contract and storage addresses, and value.
type SetState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
	ValueIndex    uint64 // encoded storage value
}

// Return set-state identifier
func (op *SetState) GetOpId() byte {
	return SetStateID
}

// Create a new set-state operation.
func NewSetState(cIdx uint32, sIdx uint32, vIdx uint64) *SetState {
	return &SetState{ContractIndex: cIdx, StorageIndex: sIdx, ValueIndex: vIdx}
}

// Read set-state operation from a file.
func ReadSetState(file *os.File) (Operation, error) {
	data := new(SetState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a set-state operation in binary format to a file.
func (op *SetState) WriteOperation(f *os.File) {
	var data = []any{op.ContractIndex, op.StorageIndex, op.ValueIndex}
	writeSlice(f, data)
}

// Execute set-state operation.
func (op *SetState) Execute(db state.StateDB, ctx *DictionaryContext) {
	contract := ctx.getContract(op.ContractIndex)
	storage := ctx.getStorage(op.StorageIndex)
	value := ctx.getValue(op.ValueIndex)
	db.SetState(contract, storage, value)
}

// Print a debug message.
func (op *SetState) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\t value: %v\n",
		ctx.getContract(op.ContractIndex),
		ctx.getStorage(op.StorageIndex),
		ctx.getValue(op.ValueIndex))
}

////////////////////////////////////////////////////////////
// GetCommittedState Operation
////////////////////////////////////////////////////////////

// GetCommittedState datastructure with encoded contract and storage addresses.
type GetCommittedState struct {
	ContractIndex uint32 // encoded contract address
	StorageIndex  uint32 // encoded storage address
}

// Return get commited-state-operation identifier.
func (op *GetCommittedState) GetOpId() byte {
	return GetCommittedStateID
}

// Create a new get-commited-state operation.
func NewGetCommittedState(cIdx uint32, sIdx uint32) *GetCommittedState {
	return &GetCommittedState{ContractIndex: cIdx, StorageIndex: sIdx}
}

// Read get-commited-state operation from a file.
func ReadGetCommittedState(file *os.File) (Operation, error) {
	data := new(GetCommittedState)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a get-commited-state operation in binary format to file.
func (op *GetCommittedState) WriteOperation(f *os.File) {
	var data = []any{op.ContractIndex, op.StorageIndex}
	writeSlice(f, data)
}

// Execute get-committed-state operation.
func (op *GetCommittedState) Execute(db state.StateDB, ctx *DictionaryContext) {
	contract := ctx.getContract(op.ContractIndex)
	storage := ctx.getStorage(op.StorageIndex)
	db.GetCommittedState(contract, storage)
}

// Print details of get-committed-state operation
func (op *GetCommittedState) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tcontract: %v\t storage: %v\n",
		ctx.getContract(op.ContractIndex),
		ctx.getStorage(op.StorageIndex))
}

////////////////////////////////////////////////////////////
// Snapshot Operation
////////////////////////////////////////////////////////////

// Snapshot datastructure with returned snapshot id
type Snapshot struct {
}

// Return snapshot operation identifier.
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

// Write the snapshot operation in binary format to file.
func (op *Snapshot) WriteOperation(f *os.File) {
}

// Execute the snapshot operation.
func (op *Snapshot) Execute(db state.StateDB, ctx *DictionaryContext) {
	db.Snapshot()
}

// Print the details for the snapshot operation.
func (op *Snapshot) Debug(*DictionaryContext) {
}

////////////////////////////////////////////////////////////
// RevertToSnapshot Operation
////////////////////////////////////////////////////////////

// Revert-to-snapshot operation's datastructure with returned snapshot id
type RevertToSnapshot struct {
	SnapshotID int
}

// Return revert-to-snapshot operation identifier.
func (op *RevertToSnapshot) GetOpId() byte {
	return RevertToSnapshotID
}

// Create a new revert-to-snapshot operation.
func NewRevertToSnapshot(SnapshotID int) *RevertToSnapshot {
	return &RevertToSnapshot{SnapshotID: SnapshotID}
}

// Read a revert-to-snapshot operation in binary format from file.
func ReadRevertToSnapshot(file *os.File) (Operation, error) {
	var data int32
	err := binary.Read(file, binary.LittleEndian, &data)
	op := &RevertToSnapshot{SnapshotID: int(data)}
	return op, err
}

// Write a revert-to-snapshot operation in binary format to file.
func (op *RevertToSnapshot) WriteOperation(f *os.File) {
	var data = []any{int32(op.SnapshotID)}
	writeSlice(f, data)
}

// Execute revert-to-snapshot operation.
func (op *RevertToSnapshot) Execute(db state.StateDB, ctx *DictionaryContext) {
	db.RevertToSnapshot(op.SnapshotID)
}

// Print a debug message for revert-to-snapshot operation.
func (op *RevertToSnapshot) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tsnapshot id: %v\n", op.SnapshotID)
}

////////////////////////////////////////////////////////////
// CreateAccount Operation
////////////////////////////////////////////////////////////

// Create-account operation's datastructure with returned snapshot id
type CreateAccount struct {
	ContractIndex uint32 // encoded contract address
}

// Return create-account operation identifier.
func (op *CreateAccount) GetOpId() byte {
	return CreateAccountID
}

// Create a new create account operation.
func NewCreateAccount(cIdx uint32) *CreateAccount {
	return &CreateAccount{ContractIndex: cIdx}
}

// Read create account operation in binary format from a file.
func ReadCreateAccount(file *os.File) (Operation, error) {
	data := new(CreateAccount)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a create account operation in binary format to file.
func (op *CreateAccount) WriteOperation(f *os.File) {
	var data = []any{op.ContractIndex}
	writeSlice(f, data)
}

// Execute create account operation.
func (op *CreateAccount) Execute(db state.StateDB, ctx *DictionaryContext) {
	contract := ctx.getContract(op.ContractIndex)
	db.CreateAccount(contract)
}

// Print a debug message for snapshot operation.
func (op *CreateAccount) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.getContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// GetBalance Operation
////////////////////////////////////////////////////////////

// GetBalance datastructure with returned snapshot id
type GetBalance struct {
	ContractIndex uint32
}

// Return snapshot operation identifier.
func (op *GetBalance) GetOpId() byte {
	return GetBalanceID
}

// Create a new snapshot operation.
func NewGetBalance(cIdx uint32) *GetBalance {
	return &GetBalance{ContractIndex: cIdx}
}

// Read snapshot operation from a file.
func ReadGetBalance(file *os.File) (Operation, error) {
	data := new(GetBalance)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a snapshot operation.
func (op *GetBalance) WriteOperation(f *os.File) {
	var data = []any{op.ContractIndex}
	writeSlice(f, data)
}

// Execute state operation
func (op *GetBalance) Execute(db state.StateDB, ctx *DictionaryContext) {
	contract := ctx.getContract(op.ContractIndex)
	db.GetBalance(contract)
}

// Print a debug message
func (op *GetBalance) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.getContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// GetCodeHash Operation
////////////////////////////////////////////////////////////

// GetCodeHash datastructure with returned snapshot id
type GetCodeHash struct {
	ContractIndex uint32
}

// Return snapshot operation identifier.
func (op *GetCodeHash) GetOpId() byte {
	return GetCodeHashID
}

// Create a new snapshot operation.
func NewGetCodeHash(cIdx uint32) *GetCodeHash {
	return &GetCodeHash{ContractIndex: cIdx}
}

// Read snapshot operation from a file.
func ReadGetCodeHash(file *os.File) (Operation, error) {
	data := new(GetCodeHash)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a snapshot operation.
func (op *GetCodeHash) WriteOperation(f *os.File) {
	var data = []any{op.ContractIndex}
	writeSlice(f, data)
}

// Execute state operation
func (op *GetCodeHash) Execute(db state.StateDB, ctx *DictionaryContext) {
	contract := ctx.getContract(op.ContractIndex)
	db.GetCodeHash(contract)
}

// Print a debug message
func (op *GetCodeHash) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.getContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// Suicide Operation
////////////////////////////////////////////////////////////

// Suicide datastructure with returned snapshot id
type Suicide struct {
	ContractIndex uint32
}

// Return snapshot operation identifier.
func (op *Suicide) GetOpId() byte {
	return SuicideID
}

// Create a new snapshot operation.
func NewSuicide(cIdx uint32) *Suicide {
	return &Suicide{ContractIndex: cIdx}
}

// Read snapshot operation from a file.
func ReadSuicide(file *os.File) (Operation, error) {
	data := new(Suicide)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a snapshot operation.
func (op *Suicide) WriteOperation(f *os.File) {
	var data = []any{op.ContractIndex}
	writeSlice(f, data)
}

// Execute state operation
func (op *Suicide) Execute(db state.StateDB, ctx *DictionaryContext) {
	contract := ctx.getContract(op.ContractIndex)
	db.Suicide(contract)
}

// Print a debug message
func (op *Suicide) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.getContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// Exist Operation
////////////////////////////////////////////////////////////

// Exist datastructure with returned snapshot id
type Exist struct {
	ContractIndex uint32
}

// Return snapshot operation identifier.
func (op *Exist) GetOpId() byte {
	return ExistID
}

// Create a new snapshot operation.
func NewExist(cIdx uint32) *Exist {
	return &Exist{ContractIndex: cIdx}
}

// Read snapshot operation from a file.
func ReadExist(file *os.File) (Operation, error) {
	data := new(Exist)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a snapshot operation.
func (op *Exist) WriteOperation(f *os.File) {
	var data = []any{op.ContractIndex}
	writeSlice(f, data)
}

// Execute state operation
func (op *Exist) Execute(db state.StateDB, ctx *DictionaryContext) {
	contract := ctx.getContract(op.ContractIndex)
	db.Exist(contract)
}

// Print a debug message
func (op *Exist) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tcontract: %v\n", ctx.getContract(op.ContractIndex))
}

////////////////////////////////////////////////////////////
// Finalise Operation
////////////////////////////////////////////////////////////

// Finalise datastructure with returned snapshot id
type Finalise struct {
	DeleteEmptyObjects bool // encoded contract address
}

// Return snapshot operation identifier.
func (op *Finalise) GetOpId() byte {
	return FinaliseID
}

// Create a new snapshot operation.
func NewFinalise(deleteEmptyObjects bool) *Finalise {
	return &Finalise{DeleteEmptyObjects: deleteEmptyObjects}
}

// Read snapshot operation from a file.
func ReadFinalise(file *os.File) (Operation, error) {
	data := new(Finalise)
	err := binary.Read(file, binary.LittleEndian, data)
	return data, err
}

// Write a snapshot operation.
func (op *Finalise) WriteOperation(f *os.File) {
	var data = []any{op.DeleteEmptyObjects}
	writeSlice(f, data)
}

// Execute state operation
func (op *Finalise) Execute(db state.StateDB, ctx *DictionaryContext) {
	db.Finalise(op.DeleteEmptyObjects)
}

// Print a debug message
func (op *Finalise) Debug(ctx *DictionaryContext) {
	fmt.Printf("\tdelete empty objects: %v\n",op.DeleteEmptyObjects)
}

////////////////////////////////////////////////////////////
// End of transaction Operation
////////////////////////////////////////////////////////////

// End-transaction operation's datastructure
type EndTransaction struct {
}

// Return end-transaction operation identifier.
func (op *EndTransaction) GetOpId() byte {
	return EndTransactionID
}

// Create a new end-transaction operation.
func NewEndTransaction() *EndTransaction {
	return &EndTransaction{}
}

// Read snapshot operation in binary format from a file.
func ReadEndTransaction(file *os.File) (Operation, error) {
	return NewEndTransaction(), nil
}

// Write a end-transaction operation in binary format to file.
func (op *EndTransaction) WriteOperation(f *os.File) {
}

// Execute end-transaction operation.
func (op *EndTransaction) Execute(db state.StateDB, ctx *DictionaryContext) {
}

// Print a debug message for end-transaction.
func (op *EndTransaction) Debug(*DictionaryContext) {
}
