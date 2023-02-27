package dict

import (
	"log"
	"math"

	"github.com/ethereum/go-ethereum/common"
)

// InvalidContractIndex used to indicate that the previously used contract index is not valid.
const InvalidContractIndex = math.MaxUint32

// DictionaryContext is a Facade for all dictionaries used to encode/decode contract/storage
// addresss, values, and snapshots.
type DictionaryContext struct {
	ContractDictionary *Dictionary[common.Address] // dictionary to compact contract addresses
	PrevContractIndex  uint32                      // previously used contract index

	StorageDictionary *Dictionary[common.Hash] // dictionary to compact storage addresses
	StorageIndexCache *IndexCache              // storage address cache

	ValueDictionary *Dictionary[common.Hash] // dictionary to compact storage values

	CodeDictionary *Dictionary[string] // dictionary to compact the bytecode of contracts

	SnapshotIndex *SnapshotIndex // snapshot index for execution (not for recording/replaying)
}

// NewDictionaryContext creates a new dictionary context.
func NewDictionaryContext() *DictionaryContext {
	return &DictionaryContext{
		ContractDictionary: NewDictionary[common.Address](),
		PrevContractIndex:  InvalidContractIndex,
		StorageDictionary:  NewDictionary[common.Hash](),
		StorageIndexCache:  NewIndexCache(),
		ValueDictionary:    NewDictionary[common.Hash](),
		CodeDictionary:     NewDictionary[string](),
		SnapshotIndex:      NewSnapshotIndex(),
	}
}

////////////////////////////////////////////////////////////////
// I/O
////////////////////////////////////////////////////////////////

// DictionaryContextDir is the dictionaries' directory of the context.
var DictionaryContextDir string = "./"

const (
	ContractMagic = 4711
	StorageMagic  = 4712
	ValueMagic    = 4713
	CodeMagic     = 4714
)

// ReadDictionaryContext reads dictionaries from files and creates a new dictionary context.
func ReadDictionaryContext() *DictionaryContext {
	ctx := NewDictionaryContext()
	err := ctx.ContractDictionary.Read(DictionaryContextDir+"contract-dictionary.dat", ContractMagic)
	if err != nil {
		log.Fatalf("Cannot read contract dictionary. Error: %v", err)
	}
	err = ctx.StorageDictionary.Read(DictionaryContextDir+"storage-dictionary.dat", StorageMagic)
	if err != nil {
		log.Fatalf("Cannot read storage dictionary. Error: %v", err)
	}
	err = ctx.ValueDictionary.Read(DictionaryContextDir+"value-dictionary.dat", ValueMagic)
	if err != nil {
		log.Fatalf("Cannot read value dictionary. Error: %v", err)
	}
	err = ctx.CodeDictionary.ReadString(DictionaryContextDir+"code-dictionary.dat", CodeMagic)
	if err != nil {
		log.Fatalf("Cannot read code dictionary. Error: %v", err)
	}
	return ctx
}

// Write dictionary context to files.
func (ctx *DictionaryContext) Write() {
	err := ctx.ContractDictionary.Write(DictionaryContextDir+"contract-dictionary.dat", ContractMagic)
	if err != nil {
		log.Fatalf("Cannot write contract dictionary. Error: %v", err)
	}
	err = ctx.StorageDictionary.Write(DictionaryContextDir+"storage-dictionary.dat", StorageMagic)
	if err != nil {
		log.Fatalf("Cannot write storage dictionary. Error: %v", err)
	}
	err = ctx.ValueDictionary.Write(DictionaryContextDir+"value-dictionary.dat", ValueMagic)
	if err != nil {
		log.Fatalf("Cannot write value dictionary. Error: %v", err)
	}
	err = ctx.CodeDictionary.WriteString(DictionaryContextDir+"code-dictionary.dat", CodeMagic)
	if err != nil {
		log.Fatalf("Cannot write code dictionary. Error: %v", err)
	}
}

////////////////////////////////////////////////////////////////
// Contract methods
////////////////////////////////////////////////////////////////

// EncodeContract encodes a given contract address and returns a contract index.
func (ctx *DictionaryContext) EncodeContract(contract common.Address) uint32 {
	cIdx, err := ctx.ContractDictionary.Encode(contract)
	if err != nil {
		log.Fatalf("Contract address could not be encoded. Error: %v", err)
	}
	if cIdx < 0 || cIdx > math.MaxUint32 {
		log.Fatalf("Contract index space depleted.", err)
	}
	ctx.PrevContractIndex = uint32(cIdx)
	return uint32(cIdx)
}

// DecodeContract decodes the contract address.
func (ctx *DictionaryContext) DecodeContract(cIdx uint32) common.Address {
	contract, err := ctx.ContractDictionary.Decode(int(cIdx))
	if err != nil {
		log.Fatalf("Contract index could not be decoded. Error: %v", err)
	}
	ctx.PrevContractIndex = cIdx
	return contract
}

// LastContractAddress returns the previously used contract address.
func (ctx *DictionaryContext) LastContractAddress() common.Address {
	if ctx.PrevContractIndex == InvalidContractIndex {
		log.Fatalf("Last contract address undefined")
	}
	return ctx.DecodeContract(ctx.PrevContractIndex)
}

////////////////////////////////////////////////////////////////
// Storage methods
////////////////////////////////////////////////////////////////

// EndcodeStorage encodes a storage address and returns an index.
func (ctx *DictionaryContext) EncodeStorage(storage common.Hash) (uint32, int) {
	sIdx, err := ctx.StorageDictionary.Encode(storage)
	if err != nil {
		log.Fatalf("Storage address could not be encoded. Error: %v", err)
	}
	if sIdx < 0 || sIdx > math.MaxUint32 {
		log.Fatalf("Contract index space depleted.", err)
	}
	pos := ctx.StorageIndexCache.Place(uint32(sIdx))
	return uint32(sIdx), pos
}

// DecodeStorage decodes a storage address.
func (ctx *DictionaryContext) DecodeStorage(sIdx uint32) common.Hash {
	storage, err := ctx.StorageDictionary.Decode(int(sIdx))
	if err != nil {
		log.Fatalf("Storage index could not be decoded. Error: %v", err)
	}
	ctx.StorageIndexCache.Place(sIdx)
	return storage
}

// ReadStorage reads index-cache and returns the storage address.
// TODO: rename method
func (ctx *DictionaryContext) ReadStorage(sPos int) common.Hash {
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
func (ctx *DictionaryContext) LookupStorage(sPos int) common.Hash {
	sIdx, err := ctx.StorageIndexCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be looked up. Error: %v", err)
	}
	return ctx.DecodeStorage(sIdx)
}

////////////////////////////////////////////////////////////////
// Value methods
////////////////////////////////////////////////////////////////

// EncodeValue encodes a value and returns an index.
func (ctx *DictionaryContext) EncodeValue(value common.Hash) uint64 {
	vIdx, err := ctx.ValueDictionary.Encode(value)
	if err != nil {
		log.Fatalf("Storage value could not be encoded. Error: %v", err)
	}
	return uint64(vIdx)
}

// DecodeValue decodes a value.
func (ctx *DictionaryContext) DecodeValue(vIdx uint64) common.Hash {
	value, err := ctx.ValueDictionary.Decode(int(vIdx))
	if err != nil {
		log.Fatalf("Value index could not be decoded. Error: %v", err)
	}
	return value
}

////////////////////////////////////////////////////////////////
// Snapshot methods
////////////////////////////////////////////////////////////////

// InitSnapshot initializes snaphot map.
func (ctx *DictionaryContext) InitSnapshot() {
	ctx.SnapshotIndex.Init()
}

// AddSnapshot adds map between recorded/replayed snapshot-id.
func (ctx *DictionaryContext) AddSnapshot(recordedID int32, replayedID int32) {
	ctx.SnapshotIndex.Add(recordedID, replayedID)
}

// GetSnapshot gets snaphot-id.
func (ctx *DictionaryContext) GetSnapshot(recordedID int32) int32 {
	replayedID, err := ctx.SnapshotIndex.Get(recordedID)
	if err != nil {
		log.Fatalf("Replayed Snapshot ID is missing. Error: %v", err)
	}
	return replayedID
}

////////////////////////////////////////////////////////////////
// Code methods
////////////////////////////////////////////////////////////////

// EncodeCode encodes byte-code to an index.
func (ctx *DictionaryContext) EncodeCode(code []byte) uint32 {
	bcIdx, err := ctx.CodeDictionary.Encode(string(code))
	if err != nil {
		log.Fatalf("Byte-code could not be encoded. Error: %v", err)
	}
	if bcIdx < 0 || bcIdx > math.MaxUint32 {
		log.Fatalf("Contract index space depleted.", err)
	}
	return uint32(bcIdx)
}

// DecodeCode decodes byte-code from an index.
func (ctx *DictionaryContext) DecodeCode(bcIdx uint32) []byte {
	code, err := ctx.CodeDictionary.Decode(int(bcIdx))
	if err != nil {
		log.Fatalf("Byte-code index could not be decoded. Error: %v", err)
	}
	return []byte(code)
}

// HasEncodedContract checks whether given address has already been inserted into dictionary
func (ctx *DictionaryContext) HasEncodedContract(addr common.Address) bool {
	_, f := ctx.ContractDictionary.valueToIdx[addr]
	return f
}

// HasEncodedStorage checks whether given storage has already been inserted into dictionary
func (ctx *DictionaryContext) HasEncodedStorage(key common.Hash) bool {
	_, f := ctx.StorageDictionary.valueToIdx[key]
	return f
}

// HasEncodedValue checks whether given value has already been inserted into dictionary
func (ctx *DictionaryContext) HasEncodedValue(value common.Hash) bool {
	_, f := ctx.ValueDictionary.valueToIdx[value]
	return f
}
