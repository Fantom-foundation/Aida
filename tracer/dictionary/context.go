package dictionary

import (
	"log"
	"math"

	"github.com/ethereum/go-ethereum/common"
)

// InvalidContractIndex used to indicate that the previously used index is not valid.
const InvalidContractIndex = math.MaxUint32

// Context is a facade for encoding/decoding contract/storage addressses, byte-code, and snapshots.
type Context struct {
	contract        *Dictionary[common.Address] // contract address dictionary
	prevContractIdx uint32                      // previously used contract index
	storage         *Dictionary[common.Hash]    // storage key dictionary
	storageCache    *IndexCache                 // storage address cache
	code            *Dictionary[string]         // bytecode dictionary
	snapshot        *SnapshotIndex              // snapshot translation table for replay
}

// NewContext creates a new dictionary context.
func NewContext() *Context {
	return &Context{
		contract:        NewDictionary[common.Address](),
		prevContractIdx: InvalidContractIndex,
		storage:         NewDictionary[common.Hash](),
		storageCache:    NewIndexCache(),
		code:            NewDictionary[string](),
		snapshot:        NewSnapshotIndex(),
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
	ContractMagic = 4711
	StorageMagic  = 4712
	CodeMagic     = 4714
)

// ReadContext reads dictionaries from files and creates a new dictionary context.
func ReadContext() *Context {
	ctx := NewContext()
	log.Printf("Read dictionary context ...")
	err := ctx.contract.Read(ContextDir+"contract-dictionary.dat", ContractMagic)
	if err != nil {
		log.Fatalf("Cannot read contract dictionary. Error: %v", err)
	}
	log.Printf("Read %v dictionary contract addresses from file.", ctx.contract.Size())
	err = ctx.storage.Read(ContextDir+"storage-dictionary.dat", StorageMagic)
	if err != nil {
		log.Fatalf("Cannot read storage dictionary. Error: %v", err)
	}
	log.Printf("Read %v dictionary storage keys from file.", ctx.storage.Size())
	err = ctx.code.ReadString(ContextDir+"code-dictionary.dat", CodeMagic)
	if err != nil {
		log.Fatalf("Cannot read code dictionary. Error: %v", err)
	}
	log.Printf("Read %v dictionary smart contracts from file.", ctx.code.Size())
	return ctx
}

// Write dictionary context to files.
func (ctx *Context) Write() {
	err := ctx.contract.Write(ContextDir+"contract-dictionary.dat", ContractMagic)
	if err != nil {
		log.Fatalf("Cannot write contract dictionary. Error: %v", err)
	}
	err = ctx.storage.Write(ContextDir+"storage-dictionary.dat", StorageMagic)
	if err != nil {
		log.Fatalf("Cannot write storage dictionary. Error: %v", err)
	}
	err = ctx.code.WriteString(ContextDir+"code-dictionary.dat", CodeMagic)
	if err != nil {
		log.Fatalf("Cannot write code dictionary. Error: %v", err)
	}
}

////////////////////////////////////////////////////////////////
// Contract methods
////////////////////////////////////////////////////////////////

// EncodeContract encodes a given contract address and returns a contract index.
func (ctx *Context) EncodeContract(contract common.Address) uint32 {
	cIdx, err := ctx.contract.Encode(contract)
	if err != nil {
		log.Fatalf("Contract address could not be encoded. Error: %v", err)
	}
	if cIdx < 0 || cIdx > math.MaxUint32 {
		log.Fatalf("Contract index space depleted.")
	}
	ctx.prevContractIdx = uint32(cIdx)
	return uint32(cIdx)
}

// DecodeContract decodes the contract address.
func (ctx *Context) DecodeContract(cIdx uint32) common.Address {
	contract, err := ctx.contract.Decode(cIdx)
	if err != nil {
		log.Fatalf("Contract index could not be decoded. Error: %v", err)
	}
	ctx.prevContractIdx = cIdx
	return contract
}

// PrevContract returns the previously used contract address.
func (ctx *Context) PrevContract() common.Address {
	if ctx.prevContractIdx == InvalidContractIndex {
		log.Fatalf("Last contract address undefined")
	}
	return ctx.DecodeContract(ctx.prevContractIdx)
}

// PrevContract returns the dictionary index of the previously used contract address.
func (ctx *Context) PrevContractIdx() uint32 {
	return uint32(ctx.prevContractIdx)
}

// HasEncodedContract checks whether a given contract address has already been inserted into dictionary
func (ctx *Context) HasEncodedContract(addr common.Address) bool {
	_, f := ctx.contract.valueToIdx[addr]
	return f
}

////////////////////////////////////////////////////////////////
// Storage methods
////////////////////////////////////////////////////////////////

// EndcodeStorage encodes a storage key and returns an index.
func (ctx *Context) EncodeStorage(storage common.Hash) (uint32, int) {
	sIdx, err := ctx.storage.Encode(storage)
	if err != nil {
		log.Fatalf("Storage address could not be encoded. Error: %v", err)
	}
	if sIdx < 0 || sIdx > math.MaxUint32 {
		log.Fatalf("Storage index space depleted.")
	}
	pos := ctx.storageCache.Place(uint32(sIdx))
	return uint32(sIdx), pos
}

// DecodeStorage decodes a storage address.
func (ctx *Context) DecodeStorage(sIdx uint32) common.Hash {
	storage, err := ctx.storage.Decode(sIdx)
	if err != nil {
		log.Fatalf("Storage index could not be decoded. Error: %v", err)
	}
	ctx.storageCache.Place(sIdx)
	return storage
}

// HasEncodedStorage checks whether a given storage key has already been inserted into dictionary
func (ctx *Context) HasEncodedStorage(key common.Hash) bool {
	_, f := ctx.storage.valueToIdx[key]
	return f
}

// DecodeStorageCache reads from cache with updating index cache.
func (ctx *Context) DecodeStorageCache(sPos int) common.Hash {
	sIdx, err := ctx.storageCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be looked up. Error: %v", err)
	}
	return ctx.DecodeStorage(sIdx)
}

// ReadStorageCache reads from cache without updating index cache.
func (ctx *Context) ReadStorageCache(sPos int) common.Hash {
	sIdx, err := ctx.storageCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be found. Error: %v", err)
	}
	storage, err := ctx.storage.Decode(sIdx)
	if err != nil {
		log.Fatalf("Storage index could not be decoded. Error: %v", err)
	}
	return storage
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
