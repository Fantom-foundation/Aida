package dict

import (
	"github.com/ethereum/go-ethereum/common"
	"log"
)

// DictionaryContext is a Facade for all dictionaries used
// to encode/decode state operations on file.
type DictionaryContext struct {
	ContractDictionary *ContractDictionary // dictionary to compact contract addresses
	StorageDictionary  *StorageDictionary  // dictionary to compact storage addresses
	ValueDictionary    *ValueDictionary    // dictionary to compact storage values

	SnapshotIndex *SnapshotIndex // Snapshot index for execution (not for recording/replaying)
}

// Create new dictionary context.
func NewDictionaryContext() *DictionaryContext {
	return &DictionaryContext{
		ContractDictionary: NewContractDictionary(),
		StorageDictionary:  NewStorageDictionary(),
		ValueDictionary:    NewValueDictionary(),
		SnapshotIndex:      NewSnapshotIndex()}
}

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

// Encode a given contract address and return a contract index.
func (ctx *DictionaryContext) EncodeContract(contract common.Address) uint32 {
	cIdx, err := ctx.ContractDictionary.Encode(contract)
	if err != nil {
		log.Fatalf("Contract address could not be encoded. Error: %v", err)
	}
	return cIdx
}

// Endcode a given storage address and retrun a storage address index.
func (ctx *DictionaryContext) EncodeStorage(storage common.Hash) uint32 {
	sIdx, err := ctx.StorageDictionary.Encode(storage)
	if err != nil {
		log.Fatalf("Storage address could not be encoded. Error: %v", err)
	}
	return sIdx
}

// Encode a storage value and return a value index.
func (ctx *DictionaryContext) EncodeValue(value common.Hash) uint64 {
	vIdx, err := ctx.ValueDictionary.Encode(value)
	if err != nil {
		log.Fatalf("Storage value could not be encoded. Error: %v", err)
	}
	return vIdx
}

// Decode the contract address for a given index.
func (ctx *DictionaryContext) DecodeContract(cIdx uint32) common.Address {
	contract, err := ctx.ContractDictionary.Decode(cIdx)
	if err != nil {
		log.Fatalf("Contract index could not be decoded. Error: %v", err)
	}
	return contract
}

// Decode the storage address for a given index.
func (ctx *DictionaryContext) DecodeStorage(sIdx uint32) common.Hash {
	storage, err := ctx.StorageDictionary.Decode(sIdx)
	if err != nil {
		log.Fatalf("Storage index could not be decoded. Error: %v", err)
	}
	return storage
}

// Decode the storage value for a given index.
func (ctx *DictionaryContext) DecodeValue(vIdx uint64) common.Hash {
	value, err := ctx.ValueDictionary.Decode(vIdx)
	if err != nil {
		log.Fatalf("Value index could not be decoded. Error: %v", err)
	}
	return value
}

// Init snaphot map.
func (ctx *DictionaryContext) InitSnapshot() {
	ctx.SnapshotIndex.Init()
}

// Add snaphot-id mapping for execution of RevertSnapshot.
func (ctx *DictionaryContext) AddSnapshot(recordedID int32, replayedID int32) {
	err := ctx.SnapshotIndex.Add(recordedID, replayedID)
	if err != nil {
		log.Fatalf("Snapshot mapping could not be added. Error: %v", err)
	}
}

// Get snaphot-id.
func (ctx *DictionaryContext) GetSnapshot(recordedID int32) int32 {
	replayedID, err := ctx.SnapshotIndex.Get(recordedID)
	if err != nil {
		log.Fatalf("Replayed Snapshot ID is missing. Error: %v", err)
	}
	return replayedID
}
