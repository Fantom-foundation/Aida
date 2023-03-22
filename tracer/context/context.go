package context

import (
	"log"

	"github.com/ethereum/go-ethereum/common"
)

// Context is a facade for encoding/decoding contract/storage addressses, byte-code, and snapshots.
type Context struct {
	prevContract common.Address // previously used contract
	keyCache     *KeyCache      // key cache
	snapshot     *SnapshotIndex // snapshot translation table for replay
}

// NewContext creates a new context context.
func NewContext() *Context {
	return &Context{
		prevContract: common.Address{},
		keyCache:     NewKeyCache(),
		snapshot:     NewSnapshotIndex(),
	}
}

////////////////////////////////////////////////////////////////
// I/O
////////////////////////////////////////////////////////////////

// ContextDir is the dictionaries' directory of the context.
var ContextDir string = "./"

// Magic constants as file identifiers for contract address, storage key
// and byte-code index files.
const (
	CodeMagic = 4714
)

////////////////////////////////////////////////////////////////
// Contract methods
////////////////////////////////////////////////////////////////

// EncodeContract encodes a given contract address and returns contract's address.
func (ctx *Context) EncodeContract(contract common.Address) common.Address {
	ctx.prevContract = contract
	return contract
}

// DecodeContract decodes the contract address.
func (ctx *Context) DecodeContract(contract common.Address) common.Address {
	ctx.prevContract = contract
	return contract
}

// PrevContract returns the previously used contract address.
func (ctx *Context) PrevContract() common.Address {
	return ctx.prevContract
}

////////////////////////////////////////////////////////////////
// Storage methods
////////////////////////////////////////////////////////////////

// EncodeKey encodes a storage key and returns an index and the key.
func (ctx *Context) EncodeKey(key common.Hash) (common.Hash, int) {
	pos := ctx.keyCache.Place(key)
	return key, pos
}

// DecodeKey decodes a storage address.
func (ctx *Context) DecodeKey(key common.Hash) common.Hash {
	ctx.keyCache.Place(key)
	return key
}

// DecodeKeyCache reads from cache with updating index cache.
func (ctx *Context) DecodeKeyCache(sPos int) common.Hash {
	key, err := ctx.keyCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be looked up. Error: %v", err)
	}
	return ctx.DecodeKey(key)
}

// ReadKeyCache reads from cache without updating index cache.
func (ctx *Context) ReadKeyCache(sPos int) common.Hash {
	key, err := ctx.keyCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be found. Error: %v", err)
	}
	return key
}

////////////////////////////////////////////////////////////////
// Snapshot methods
////////////////////////////////////////////////////////////////

// InitSnapshot initializes snaphot map.
func (ctx *Context) InitSnapshot() {
	ctx.snapshot.Init()
}

// AddSnapshot adds map between recorded/replayed snapshot-id.
func (ctx *Context) AddSnapshot(recordedID int32, replayedID int32) {
	ctx.snapshot.Add(recordedID, replayedID)
}

// GetSnapshot gets snaphot-id.
func (ctx *Context) GetSnapshot(recordedID int32) int32 {
	replayedID, err := ctx.snapshot.Get(recordedID)
	if err != nil {
		log.Fatalf("Replayed Snapshot ID is missing. Error: %v", err)
	}
	return replayedID
}
