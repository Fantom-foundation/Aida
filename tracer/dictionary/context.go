package dictionary

import (
	"log"
	"math"

	"github.com/ethereum/go-ethereum/common"
)

// InvalidContractIndex used to indicate that the previously used contract index is not valid.
const InvalidContractIndex = math.MaxUint32

// Context is a facade for all dictionaries used to encode/decode contract/storage
// addresss, and snapshots.
type Context struct {
	ContractDictionary *Dictionary[common.Address] // dictionary to compact contract addresses
	PrevContractIndex  uint32                      // previously used contract index

	StorageDictionary *Dictionary[common.Hash] // dictionary to compact storage addresses
	StorageIndexCache *IndexCache              // storage address cache

	CodeDictionary *Dictionary[string] // dictionary to compact the bytecode of contracts

	SnapshotIndex *SnapshotIndex // snapshot index for execution (not for recording/replaying)
}

// NewContext creates a new dictionary context.
func NewContext() *Context {
	return &Context{
		ContractDictionary: NewDictionary[common.Address](),
		PrevContractIndex:  InvalidContractIndex,
		StorageDictionary:  NewDictionary[common.Hash](),
		StorageIndexCache:  NewIndexCache(),
		CodeDictionary:     NewDictionary[string](),
		SnapshotIndex:      NewSnapshotIndex(),
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
	err := ctx.ContractDictionary.Read(ContextDir+"contract-dictionary.dat", ContractMagic)
	if err != nil {
		log.Fatalf("Cannot read contract dictionary. Error: %v", err)
	}
	err = ctx.StorageDictionary.Read(ContextDir+"storage-dictionary.dat", StorageMagic)
	if err != nil {
		log.Fatalf("Cannot read storage dictionary. Error: %v", err)
	}
	err = ctx.CodeDictionary.ReadString(ContextDir+"code-dictionary.dat", CodeMagic)
	if err != nil {
		log.Fatalf("Cannot read code dictionary. Error: %v", err)
	}
	return ctx
}

// Write dictionary context to files.
func (ctx *Context) Write() {
	err := ctx.ContractDictionary.Write(ContextDir+"contract-dictionary.dat", ContractMagic)
	if err != nil {
		log.Fatalf("Cannot write contract dictionary. Error: %v", err)
	}
	err = ctx.StorageDictionary.Write(ContextDir+"storage-dictionary.dat", StorageMagic)
	if err != nil {
		log.Fatalf("Cannot write storage dictionary. Error: %v", err)
	}
	err = ctx.CodeDictionary.WriteString(ContextDir+"code-dictionary.dat", CodeMagic)
	if err != nil {
		log.Fatalf("Cannot write code dictionary. Error: %v", err)
	}
}

////////////////////////////////////////////////////////////////
// Contract methods
////////////////////////////////////////////////////////////////

// EncodeContract encodes a given contract address and returns a contract index.
func (ctx *Context) EncodeContract(contract common.Address) uint32 {
	cIdx, err := ctx.ContractDictionary.Encode(contract)
	if err != nil {
		log.Fatalf("Contract address could not be encoded. Error: %v", err)
	}
	if cIdx < 0 || cIdx > math.MaxUint32 {
		log.Fatalf("Contract index space depleted.")
	}
	ctx.PrevContractIndex = uint32(cIdx)
	return uint32(cIdx)
}

// DecodeContract decodes the contract address.
func (ctx *Context) DecodeContract(cIdx uint32) common.Address {
	contract, err := ctx.ContractDictionary.Decode(int(cIdx))
	if err != nil {
		log.Fatalf("Contract index could not be decoded. Error: %v", err)
	}
	ctx.PrevContractIndex = cIdx
	return contract
}

// LastContractAddress returns the previously used contract address.
func (ctx *Context) LastContractAddress() common.Address {
	if ctx.PrevContractIndex == InvalidContractIndex {
		log.Fatalf("Last contract address undefined")
	}
	return ctx.DecodeContract(ctx.PrevContractIndex)
}

// HasEncodedContract checks whether a given contract address has already been inserted into dictionary
func (ctx *Context) HasEncodedContract(addr common.Address) bool {
	_, f := ctx.ContractDictionary.valueToIdx[addr]
	return f
}

////////////////////////////////////////////////////////////////
// Storage methods
////////////////////////////////////////////////////////////////

// EndcodeStorage encodes a storage key and returns an index.
func (ctx *Context) EncodeStorage(storage common.Hash) (uint32, int) {
	sIdx, err := ctx.StorageDictionary.Encode(storage)
	if err != nil {
		log.Fatalf("Storage address could not be encoded. Error: %v", err)
	}
	if sIdx < 0 || sIdx > math.MaxUint32 {
		log.Fatalf("Storage index space depleted.")
	}
	pos := ctx.StorageIndexCache.Place(uint32(sIdx))
	return uint32(sIdx), pos
}

// DecodeStorage decodes a storage address.
func (ctx *Context) DecodeStorage(sIdx uint32) common.Hash {
	storage, err := ctx.StorageDictionary.Decode(int(sIdx))
	if err != nil {
		log.Fatalf("Storage index could not be decoded. Error: %v", err)
	}
	ctx.StorageIndexCache.Place(sIdx)
	return storage
}

// ReadStorage reads index-cache and returns the storage address.
// TODO: rename method
func (ctx *Context) ReadStorage(sPos int) common.Hash {
	sIdx, err := ctx.StorageIndexCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be found. Error: %v", err)
	}
	storage, err := ctx.StorageDictionary.Decode(int(sIdx))
	if err != nil {
		log.Fatalf("Storage index could not be decoded. Error: %v", err)
	}
	return storage
}

// LookupStorage reads and updates index-cache.
// TODO: rename method
func (ctx *Context) LookupStorage(sPos int) common.Hash {
	sIdx, err := ctx.StorageIndexCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be looked up. Error: %v", err)
	}
	return ctx.DecodeStorage(sIdx)
}

// HasEncodedStorage checks whether a given storage key has already been inserted into dictionary
func (ctx *Context) HasEncodedStorage(key common.Hash) bool {
	_, f := ctx.StorageDictionary.valueToIdx[key]
	return f
}

////////////////////////////////////////////////////////////////
// Snapshot methods
////////////////////////////////////////////////////////////////

// InitSnapshot initializes snaphot map.
func (ctx *Context) InitSnapshot() {
	ctx.SnapshotIndex.Init()
}

// AddSnapshot adds map between recorded/replayed snapshot-id.
func (ctx *Context) AddSnapshot(recordedID int32, replayedID int32) {
	ctx.SnapshotIndex.Add(recordedID, replayedID)
}

// GetSnapshot gets snaphot-id.
func (ctx *Context) GetSnapshot(recordedID int32) int32 {
	replayedID, err := ctx.SnapshotIndex.Get(recordedID)
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
	bcIdx, err := ctx.CodeDictionary.Encode(string(code))
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
	code, err := ctx.CodeDictionary.Decode(int(bcIdx))
	if err != nil {
		log.Fatalf("Byte-code index could not be decoded. Error: %v", err)
	}
	return []byte(code)
}
