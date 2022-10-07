package tracer

import (
	"github.com/ethereum/go-ethereum/common"
	"log"
)

////////////////////////////////////////////////////////////
// Dictionary Context
////////////////////////////////////////////////////////////

// DictionaryContext contains the contract/storage dictionaries
// so that a recorded StateDB operation can be executed.
type DictionaryContext struct {
	ContractDictionary *ContractDictionary // dictionary to compact contract addresses
	StorageDictionary  *StorageDictionary  // dictionary to compact storage addresses
	ValueDictionary    *ValueDictionary    // dictionary to compact storage values
}

// Create new dictionary context.
func NewDictionaryContext() *DictionaryContext {
	return &DictionaryContext{
		ContractDictionary: NewContractDictionary(),
		StorageDictionary:  NewStorageDictionary(),
		ValueDictionary:    NewValueDictionary()}
}

// Read dictionary context from files.
// TODO: Error handling is missing
func ReadDictionaryContext() *DictionaryContext {
	ctx := NewDictionaryContext()
	ctx.ContractDictionary.Read(TraceDir + "contract-dictionary.dat")
	ctx.StorageDictionary.Read(TraceDir + "storage-dictionary.dat")
	ctx.ValueDictionary.Read(TraceDir + "value-dictionary.dat")
	return ctx
}

// Write dictionary context to files.
// TODO: Error handling is missing
func (ctx *DictionaryContext) Write() {
	ctx.ContractDictionary.Write(TraceDir + "contract-dictionary.dat")
	ctx.StorageDictionary.Write(TraceDir + "storage-dictionary.dat")
	ctx.ValueDictionary.Write(TraceDir + "value-dictionary.dat")
}

// Get the contract address for a given index.
func (ctx *DictionaryContext) getContract(cIdx uint32) common.Address {
	contract, err := ctx.ContractDictionary.Decode(cIdx)
	if err != nil {
		log.Fatalf("Contract index could not be decoded, error: %v", err)
	}
	return contract
}

// Get the storage address for a given index.
func (ctx *DictionaryContext) getStorage(sIdx uint32) common.Hash {
	storage, err := ctx.StorageDictionary.Decode(sIdx)
	if err != nil {
		log.Fatalf("Storage index could not be decoded, error: %v", err)
	}
	return storage
}

// Get the storage value for a given index.
func (ctx *DictionaryContext) getValue(vIdx uint64) common.Hash {
	value, err := ctx.ValueDictionary.Decode(vIdx)
	if err != nil {
		log.Fatalf("Value index could not be decoded, error: %v", err)
	}
	return value
}

// Encode a given contract address and return a contract index.
func (ctx *DictionaryContext) encodeContract(contract common.Address) uint32 {
	cIdx, err := ctx.ContractDictionary.Encode(contract)
	if err != nil {
		log.Fatalf("Contract could not be encoded, error: %v", err)
	}
	return cIdx
}

// Endcode a given storage address and retrun a storage address index.
func (ctx *DictionaryContext) encodeStorage(storage common.Hash) uint32 {
	sIdx, err := ctx.StorageDictionary.Encode(storage)
	if err != nil {
		log.Fatalf("Storage could not be encoded, error: %v", err)
	}
	return sIdx
}

// Encode a storage value and return a value index.
func (ctx *DictionaryContext) encodeValue(value common.Hash) uint64 {
	vIdx, err := ctx.ValueDictionary.Encode(value)
	if err != nil {
		log.Fatalf("Value could not be encoded, error: %v", err)
	}
	return vIdx
}
