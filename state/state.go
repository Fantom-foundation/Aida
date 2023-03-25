package state

import (
	"fmt"
	"math/big"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

	Error() error
}

type StateDB interface {
	BasicStateDB

	// Requests the StateDB to flush all its content to secondary storage and shut down.
	// After this call no more operations will be allowed on the state.
	Close() error

	// stateDB handler
	BeginBlockApply() error

	// StartBulkLoad creates a interface supporting the efficient loading of large amount
	// of data as it is, for instance, needed during priming. Only one bulk load operation
	// may be active at any time and no other concurrent operations on the StateDB are
	// while it is alive.
	StartBulkLoad() BulkLoad

	// GetArchiveState creates a state instance linked to a historic block state in an
	// optionally present archive. The operation fails if there is no archive or the
	// specified block is not present in the archive.
	GetArchiveState(block uint64) (StateDB, error)

	// Requests a description of the current memory usage of this State DB. Implementations
	// not supporting this may return nil.
	GetMemoryUsage() *MemoryUsage

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
	PrepareSubstate(*substate.SubstateAlloc, uint64)

	BeginErigonExecution() func()
}

// BulkWrite is a faster interface to StateDB instances for writing data without
// the overhead of snapshots or transactions. It is mainly intended for priming DB
// instances before running evaluations.
type BulkLoad interface {
	CreateAccount(common.Address)
	SetBalance(common.Address, *big.Int)
	SetNonce(common.Address, uint64)
	SetState(common.Address, common.Hash, common.Hash)
	SetCode(common.Address, []byte)

	// Close ends the bulk insertion, finalizes the internal state, and released the
	// underlying StateDB instance for regular operations.
	Close() error
}

// A description of the memory usage of a StateDB implementation.
type MemoryUsage struct {
	UsedBytes uint64
	Breakdown fmt.Stringer
}
