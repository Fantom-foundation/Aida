package context

import (
	"log"
	"math"

	"github.com/ethereum/go-ethereum/common"
)

// Context is a facade for encoding/decoding contract/storage addressses, byte-code, and snapshots.
type Context struct {
	prevContract common.Address      // previously used contract
	keyCache     *KeyCache           // key cache
	code         *Dictionary[string] // bytecode context
	snapshot     *SnapshotIndex      // snapshot translation table for replay
}

// NewContext creates a new context context.
func NewContext() *Context {
	return &Context{
		prevContract: common.Address{},
		keyCache:     NewKeyCache(),
		code:         NewDictionary[string](),
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

// ReadContext reads dictionaries from files and creates a new context context.
func ReadContext() *Context {
	ctx := NewContext()
	err := ctx.code.ReadString(ContextDir+"code-context.dat", CodeMagic)
	if err != nil {
		log.Fatalf("Cannot read code context. Error: %v", err)
	}
	log.Printf("Read %v context smart contracts from file.", ctx.code.Size())
	return ctx
}

// Write context context to files.
func (ctx *Context) Write() {
	err := ctx.code.WriteString(ContextDir+"code-context.dat", CodeMagic)
	if err != nil {
		log.Fatalf("Cannot write code context. Error: %v", err)
	}
}

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

////////////////////////////////////////////////////////////////
// Code methods
////////////////////////////////////////////////////////////////

// EncodeCode encodes the given byte-code to an index and returns the index.
func (ctx *Context) EncodeCode(code []byte) uint32 {
	bcIdx, err := ctx.code.Encode(string(code))
	if err != nil {
		log.Fatalf("Byte-code could not be encoded. Error: %v", err)
	}
	if bcIdx < 0 || bcIdx > math.MaxUint32 {
		log.Fatalf("Byte-code index space depleted.")
	}
	return uint32(bcIdx)
}

// DecodeCode returns the byte-code for a given byte-code index.
func (ctx *Context) DecodeCode(bcIdx uint32) []byte {
	code, err := ctx.code.Decode(bcIdx)
	if err != nil {
		log.Fatalf("Byte-code index could not be decoded. Error: %v", err)
	}
	return []byte(code)
}
