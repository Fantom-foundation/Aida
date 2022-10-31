package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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

	// Transaction handling
	Snapshot() int
	RevertToSnapshot(int)
	Finalise(bool)

	// Substate specific
	GetSubstatePostAlloc() substate.SubstateAlloc
}

type StateDB interface {
	BasicStateDB

	// Requests the StateDB to flush all its content to secondary storage and shut down.
	// After this call no more operations will be allowed on the state.
	Close() error

	// ---- Optional Development & Debugging Features ----

	// Used to initiate the state DB for the next transaction.
	// This is mainly for development purposes to support in-memory DB implementations.
	PrepareSubstate(*substate.SubstateAlloc)
}
