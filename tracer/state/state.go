package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/substate"
)

type BasicStateDB interface {
	// Account management.
	CreateAccount(common.Address)
	Exist(common.Address) bool
	Empty(common.Address) bool

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// Balance
	GetBalance(common.Address) *big.Int
	AddBalance(common.Address, *big.Int)
	SubBalance(common.Address, *big.Int)

	// Nonce
	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	// State
	GetCommittedState(common.Address, common.Hash) common.Hash
	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	// Code handling.
	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	// Gas calculation
	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64

	// Access list
	PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList)
	AddressInAccessList(addr common.Address) bool
	SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool)
	AddAddressToAccessList(addr common.Address)
	AddSlotToAccessList(addr common.Address, slot common.Hash)

	// Logging
	AddLog(*types.Log)
	GetLogs(common.Hash, common.Hash) []*types.Log

	// Transaction handling
	// There are 4 layers of concepts governing the visibility of state effects:
	//  - snapshots .. enclosing (sub-)contract calls, supporting reverts (=rollbacks)
	//  - transactions .. processing a single block chain event, comprising a hierachy of contract calls
	//  - blocks .. groups of transactions, at boundaries effects become visible (and final) to API servers
	//  - epochs .. groups of blocks, at boundaries state becomes syncable between nodes

	Snapshot() int
	RevertToSnapshot(int)

	BeginTransaction(uint32)
	EndTransaction()

	BeginBlock(uint64)
	EndBlock()

	BeginEpoch(uint64)
	EndEpoch()
}

type StateDB interface {
	BasicStateDB

	// Requests the StateDB to flush all its content to secondary storage and shut down.
	// After this call no more operations will be allowed on the state.
	Close() error

	// stateDB handler
	BeginBlockApply(common.Hash) error

	// ---- Artefacts from Geth dependency ----

	// The following functions may be used by StateDB implementations for backward-compatibilty
	// and will be called accordingly by the tracer and EVM runner. However, implementations may
	// chose to ignore those.

	Prepare(common.Hash, int)
	AddPreimage(common.Hash, []byte)
	Finalise(bool)
	IntermediateRoot(bool) common.Hash
	Commit(bool) (common.Hash, error)
	ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error

	// ---- Optional Development & Debugging Features ----

	// Substate specific
	GetSubstatePostAlloc() substate.SubstateAlloc

	// Used to initiate the state DB for the next transaction.
	// This is mainly for development purposes to support in-memory DB implementations.
	PrepareSubstate(*substate.SubstateAlloc)
}
