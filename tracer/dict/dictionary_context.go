package dict

import (
	"github.com/ethereum/go-ethereum/common"
	"log"
	"math"
)

const InvalidContractIndex = math.MaxUint32

// DictionaryContext is a Facade for all dictionaries used
// to encode/decode state operations on file.
type DictionaryContext struct {
	ContractDictionary *ContractDictionary // dictionary to compact contract addresses
	PrevContractIndex  uint32              // previously used contract index
	StorageDictionary  *StorageDictionary  // dictionary to compact storage addresses
	StorageCache       *IndexCache         // storage address cache
	ValueDictionary    *ValueDictionary    // dictionary to compact storage values
	SnapshotIndex      *SnapshotIndex      // snapshot index for execution (not for recording/replaying)
}

// Create new dictionary context.
func NewDictionaryContext() *DictionaryContext {
	return &DictionaryContext{
		ContractDictionary: NewContractDictionary(),
		StorageDictionary:  NewStorageDictionary(),
		ValueDictionary:    NewValueDictionary(),
		SnapshotIndex:      NewSnapshotIndex(),
		PrevContractIndex:  InvalidContractIndex,
		StorageCache:       NewIndexCache()}
}

////////////////////////////////////////////////////////////////
// I/O
////////////////////////////////////////////////////////////////

var DictDir string = "./"

// Read dictionary context from files.
func ReadDictionaryContext() *DictionaryContext {
	ctx := NewDictionaryContext()
	err := ctx.ContractDictionary.Read(DictDir + "contract-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot read contract dictionary. Error: %v", err)
	}
	err = ctx.StorageDictionary.Read(DictDir + "storage-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot read storage dictionary. Error: %v", err)
	}
	err = ctx.ValueDictionary.Read(DictDir + "value-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot read value dictionary. Error: %v", err)
	}
	return ctx
}

// Write dictionary context to files.
func (ctx *DictionaryContext) Write() {
	err := ctx.ContractDictionary.Write(DictDir + "contract-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot write contract dictionary. Error: %v", err)
	}
	err = ctx.StorageDictionary.Write(DictDir + "storage-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot write storage dictionary. Error: %v", err)
	}
	err = ctx.ValueDictionary.Write(DictDir + "value-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot write value dictionary. Error: %v", err)
	}
}

////////////////////////////////////////////////////////////////
// Contract methods
////////////////////////////////////////////////////////////////

// Encode a given contract address and return a contract index.
func (ctx *DictionaryContext) EncodeContract(contract common.Address) uint32 {
	cIdx, err := ctx.ContractDictionary.Encode(contract)
	if err != nil {
		log.Fatalf("Contract address could not be encoded. Error: %v", err)
	}
	ctx.PrevContractIndex = cIdx
	return cIdx
}

// Decode the contract address for a given index.
func (ctx *DictionaryContext) DecodeContract(cIdx uint32) common.Address {
	contract, err := ctx.ContractDictionary.Decode(cIdx)
	if err != nil {
		log.Fatalf("Contract index could not be decoded. Error: %v", err)
	}
	ctx.PrevContractIndex = cIdx
	return contract
}

// Read the contract address for a given index.
func (ctx *DictionaryContext) LastContractAddress() common.Address {
	if ctx.PrevContractIndex == InvalidContractIndex {
		log.Fatalf("Last contract address undefined")
	}
	return ctx.DecodeContract(ctx.PrevContractIndex)
}

////////////////////////////////////////////////////////////////
// Storage methods
////////////////////////////////////////////////////////////////

// Endcode a given storage address and retrun a storage address index.
func (ctx *DictionaryContext) EncodeStorage(storage common.Hash) (uint32, int) {
	sIdx, err := ctx.StorageDictionary.Encode(storage)
	if err != nil {
		log.Fatalf("Storage address could not be encoded. Error: %v", err)
	}
	pos := ctx.StorageCache.Place(sIdx)
	return sIdx, pos
}

// Decode the storage address for a given index.
func (ctx *DictionaryContext) DecodeStorage(sIdx uint32) common.Hash {
	storage, err := ctx.StorageDictionary.Decode(sIdx)
	if err != nil {
		log.Fatalf("Storage index could not be decoded. Error: %v", err)
	}
	ctx.StorageCache.Place(sIdx)
	return storage
}

// Read the storage address for a given index.
func (ctx *DictionaryContext) ReadStorage(sPos int) common.Hash {
	sIdx, err := ctx.StorageCache.Lookup(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be found. Error: %v", err)
	}
	storage, err := ctx.StorageDictionary.Decode(sIdx)
	if err != nil {
		log.Fatalf("Storage index could not be decoded. Error: %v", err)
	}
	return storage
}

// Look up the storage address for a given index.
func (ctx *DictionaryContext) LookupStorage(sPos int) common.Hash {
	sIdx, err := ctx.StorageCache.Lookup(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be found. Error: %v", err)
	}
	return ctx.DecodeStorage(sIdx)
}

////////////////////////////////////////////////////////////////
// Value methods
////////////////////////////////////////////////////////////////

// Encode a storage value and return a value index.
func (ctx *DictionaryContext) EncodeValue(value common.Hash) uint64 {
	vIdx, err := ctx.ValueDictionary.Encode(value)
	if err != nil {
		log.Fatalf("Storage value could not be encoded. Error: %v", err)
	}
	return vIdx
}

// Decode the storage value for a given index.
func (ctx *DictionaryContext) DecodeValue(vIdx uint64) common.Hash {
	value, err := ctx.ValueDictionary.Decode(vIdx)
	if err != nil {
		log.Fatalf("Value index could not be decoded. Error: %v", err)
	}
	return value
}

////////////////////////////////////////////////////////////////
// Snapshot methods
////////////////////////////////////////////////////////////////

// Init snaphot map.
func (ctx *DictionaryContext) InitSnapshot() {
	ctx.SnapshotIndex.Init()
}

// Add snaphot-id mapping for execution of RevertSnapshot.
func (ctx *DictionaryContext) AddSnapshot(recordedID uint16, replayedID uint16) {
	ctx.SnapshotIndex.Add(recordedID, replayedID)
}

// Get snaphot-id.
func (ctx *DictionaryContext) GetSnapshot(recordedID uint16) uint16 {
	replayedID, err := ctx.SnapshotIndex.Get(recordedID)
	if err != nil {
		log.Fatalf("Replayed Snapshot ID is missing. Error: %v", err)
	}
	return replayedID
}

////////////////////////////////////////////////////////////////
// Index cache methods
////////////////////////////////////////////////////////////////

// Clear count queues.
func (ctx *DictionaryContext) ClearIndexCaches() {
	ctx.PrevContractIndex = InvalidContractIndex
	ctx.StorageCache.ClearIndexCache()
}
