package state

//go:generate mockgen -source state.go -destination state_mocks.go -package state

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// VmStateDB is the basic StateDB interface required by the EVM and related
// transaction processing components for interacting with the StateDB.
type VmStateDB interface {
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
	//  - transactions .. processing a single block chain event, comprising a hierarchy of contract calls
	//  - blocks .. groups of transactions, at boundaries effects become visible (and final) to API servers
	//  - sync-periods .. groups of blocks, at boundaries state becomes synchronizable between nodes

	Snapshot() int
	RevertToSnapshot(int)

	BeginTransaction(uint32) error
	EndTransaction() error

	// ---- Artifacts from Geth dependency ----

	// The following functions may be used by the Geth implementations for backward-compatibility
	// and will be called accordingly by the tracer and EVM runner. However, implementations may
	// chose to ignore those.

	Prepare(common.Hash, int)
	AddPreimage(common.Hash, []byte)
	ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error

	// ---- Optional Development & Debugging Features ----

	// Substate specific
	GetSubstatePostAlloc() txcontext.WorldState
}

// NonCommittableStateDB is an extension of the VmStateDB interface and is intended
// to serve as the type used for referencing immutable historical state, in particular
// state views obtained from Archive DBs. While transaction-local updates and
// modifications are allowed, the interface does not provide means for persisting those
// changes (=commit them).
type NonCommittableStateDB interface {
	VmStateDB

	// GetHash obtains a cryptographic hash certifying the committed content of the
	// represented state. It does not consider any temporary modifications conducted
	// through the VmStateDB interface on the state.
	GetHash() (common.Hash, error)

	// Release frees resources bound by this view. Release should be called on every
	// instance once all operations have been completed. Once released, no further
	// operations on the respective instance are allowed.
	Release() error
}

// StateDB is an extension of the VmStateDB interface adding general DB management
// operations that are beyond the interface required by the EVM. In particular,
// this includes the handling of blocks and sync-periods, archive handling, and
// BulkLoad support.
type StateDB interface {
	VmStateDB

	BeginBlock(uint64) error
	EndBlock() error

	BeginSyncPeriod(uint64)
	EndSyncPeriod()

	// GetHash computes a comprehensive hash over a snapshot of the entire state. Note, to
	// be of any value, no concurrent modifications should be conducted while computing the
	// hash. State implementations are not required to implement any specific hash function
	// function unless specifically declared to do so. For instance, Geth and Carmen S5 are
	// expected to produce the same hashes for the same content.
	GetHash() (common.Hash, error)

	Error() error

	// Requests the StateDB to flush all its content to secondary storage and shut down.
	// After this call no more operations will be allowed on the state.
	Close() error

	// StartBulkLoad creates a interface supporting the efficient loading of large amount
	// of data as it is, for instance, needed during priming. Only one bulk load operation
	// may be active at any time and no other concurrent operations on the StateDB are
	// while it is alive. Data inserted during a bulk-load will appear as if it was inserted
	// in a single block.
	StartBulkLoad(block uint64) BulkLoad

	// GetArchiveState creates a state instance linked to a historic block state in an
	// optionally present archive. The operation fails if there is no archive or the
	// specified block is not present in the archive.
	GetArchiveState(block uint64) (NonCommittableStateDB, error)

	// GetArchiveBlockHeight provides the block height available in the archive.
	// An error is returned if the archive is not enabled or a lookup issue occurred.
	GetArchiveBlockHeight() (height uint64, empty bool, err error)

	// Requests a description of the current memory usage of this State DB. Implementations
	// not supporting this may return nil.
	GetMemoryUsage() *MemoryUsage

	// ---- Artifacts from Geth dependency ----

	// The following functions may be used by StateDB implementations for backward-compatibility
	// and will be called accordingly by the tracer and EVM runner. However, implementations may
	// chose to ignore those.

	Finalise(bool)
	IntermediateRoot(bool) common.Hash
	Commit(bool) (common.Hash, error)

	// ---- Optional Development & Debugging Features ----

	// Used to initiate the state DB for the next transaction.
	// This is mainly for development purposes to support in-memory DB implementations.
	PrepareSubstate(substate txcontext.WorldState, block uint64)

	// Used to retrieve the shadow DB (if there is one) for testing purposes so that
	// the shadow DB can be used to query state directly. If there is no shadow DB,
	// nil is returned.
	GetShadowDB() StateDB
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
