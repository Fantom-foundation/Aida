package dict

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"math"
	"os"
	"sort"
)

// InvalidContractIndex used to indicate that the previously used contract index is not valid.
const InvalidContractIndex = math.MaxUint32

// DictionaryContext is a Facade for all dictionaries used to encode/decode contract/storage
// addresss, values, and snapshots.
type DictionaryContext struct {
	ContractDictionary *ContractDictionary // dictionary to compact contract addresses
	PrevContractIndex  uint32              // previously used contract index

	StorageDictionary *StorageDictionary // dictionary to compact storage addresses
	StorageIndexCache *IndexCache        // storage address cache

	ValueDictionary *ValueDictionary // dictionary to compact storage values

	CodeDictionary *CodeDictionary // dictionary to compact the bytecode of contracts

	SnapshotIndex *SnapshotIndex // snapshot index for execution (not for recording/replaying)

	ContractFreq []uint64 // number of each operation accesses to contract
	StorageFreq  []uint64 // number of each operation accesses to storage
	ValueFreq    []uint64 // number of each operation accesses to value
	OpFreq       []uint64 // number of operation invocations
	PrevOpId     byte
	TFreq        map[[2]byte]uint64
}

// operationFrequency is used for distribution calculation of operations and their frequencies
type operationFrequency struct {
	opId      int
	frequency uint64
}

// NewDictionaryContext creates a new dictionary context.
func NewDictionaryContext() *DictionaryContext {
	return &DictionaryContext{
		ContractDictionary: NewContractDictionary(),
		PrevContractIndex:  InvalidContractIndex,
		StorageDictionary:  NewStorageDictionary(),
		StorageIndexCache:  NewIndexCache(),
		ValueDictionary:    NewValueDictionary(),
		CodeDictionary:     NewCodeDictionary(),
		SnapshotIndex:      NewSnapshotIndex(),
	}
}

// NewDictionaryContext creates a new dictionary context.
func NewDictionaryStochasticContext(beginBlockId byte, numProfiledOperations byte) *DictionaryContext {
	return &DictionaryContext{
		ContractDictionary: NewContractDictionary(),
		PrevContractIndex:  InvalidContractIndex,
		StorageDictionary:  NewStorageDictionary(),
		StorageIndexCache:  NewIndexCache(),
		ValueDictionary:    NewValueDictionary(),
		CodeDictionary:     NewCodeDictionary(),
		SnapshotIndex:      NewSnapshotIndex(),
		ContractFreq:       make([]uint64, numProfiledOperations),
		StorageFreq:        make([]uint64, numProfiledOperations),
		ValueFreq:          make([]uint64, numProfiledOperations),
		OpFreq:             make([]uint64, numProfiledOperations),
		PrevOpId:           beginBlockId,
		TFreq:              map[[2]byte]uint64{},
	}
}

////////////////////////////////////////////////////////////////
// I/O
////////////////////////////////////////////////////////////////

// DictionaryContextDir is the dictionaries' directory of the context.
var DictionaryContextDir string = "./"

// ReadDictionaryContext reads dictionaries from files and creates a new dictionary context.
func ReadDictionaryContext() *DictionaryContext {
	ctx := NewDictionaryContext()
	err := ctx.ContractDictionary.Read(DictionaryContextDir + "contract-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot read contract dictionary. Error: %v", err)
	}
	err = ctx.StorageDictionary.Read(DictionaryContextDir + "storage-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot read storage dictionary. Error: %v", err)
	}
	err = ctx.ValueDictionary.Read(DictionaryContextDir + "value-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot read value dictionary. Error: %v", err)
	}
	err = ctx.CodeDictionary.Read(DictionaryContextDir + "code-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot read code dictionary. Error: %v", err)
	}
	return ctx
}

// Write dictionary context to files.
func (ctx *DictionaryContext) Write() {
	err := ctx.ContractDictionary.Write(DictionaryContextDir + "contract-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot write contract dictionary. Error: %v", err)
	}
	err = ctx.StorageDictionary.Write(DictionaryContextDir + "storage-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot write storage dictionary. Error: %v", err)
	}
	err = ctx.ValueDictionary.Write(DictionaryContextDir + "value-dictionary.dat")
	if err != nil {
		log.Fatalf("Cannot write value dictionary. Error: %v", err)
	}
	err = ctx.CodeDictionary.Write(DictionaryContextDir + "code-dictionary.dat")
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
	ctx.PrevContractIndex = cIdx
	return cIdx
}

// DecodeContract decodes the contract address.
func (ctx *DictionaryContext) DecodeContract(cIdx uint32) common.Address {
	contract, err := ctx.ContractDictionary.Decode(cIdx)
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
	pos := ctx.StorageIndexCache.Place(sIdx)
	return sIdx, pos
}

// DecodeStorage decodes a storage address.
func (ctx *DictionaryContext) DecodeStorage(sIdx uint32) common.Hash {
	storage, err := ctx.StorageDictionary.Decode(sIdx)
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
	storage, err := ctx.StorageDictionary.Decode(sIdx)
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
		log.Fatalf("Storage position could not be found. Error: %v", err)
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
	return vIdx
}

// DecodeValue decodes a value.
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
	bcIdx, err := ctx.CodeDictionary.Encode(code)
	if err != nil {
		log.Fatalf("Byte-code could not be encoded. Error: %v", err)
	}
	return bcIdx
}

// DecodeCode decodes byte-code from an index.
func (ctx *DictionaryContext) DecodeCode(bcIdx uint32) []byte {
	code, err := ctx.CodeDictionary.Decode(bcIdx)
	if err != nil {
		log.Fatalf("Byte-code index could not be decoded. Error: %v", err)
	}
	return code
}

// HasEncodedContract checks whether given address has already been inserted into dictionary
func (ctx *DictionaryContext) HasEncodedContract(addr common.Address) bool {
	_, f := ctx.ContractDictionary.contractToIdx[addr]
	return f
}

// HasEncodedStorage checks whether given storage has already been inserted into dictionary
func (ctx *DictionaryContext) HasEncodedStorage(key common.Hash) bool {
	_, f := ctx.StorageDictionary.storageToIdx[key]
	return f
}

// HasEncodedValue checks whether given value has already been inserted into dictionary
func (ctx *DictionaryContext) HasEncodedValue(value common.Hash) bool {
	_, f := ctx.ValueDictionary.valueToIdx[value]
	return f
}

// WriteDistributions dictionary distributions into files.
func (ctx *DictionaryContext) WriteDistributions() {
	err := ctx.WriteDistribution(DictionaryContextDir+"contract-distribution.dat", ctx.ContractDictionary.frequency)
	if err != nil {
		log.Fatalf("Cannot write contract distribution. Error: %v", err)
	}
	err = ctx.WriteDistribution(DictionaryContextDir+"storage-distribution.dat", ctx.StorageDictionary.frequency)
	if err != nil {
		log.Fatalf("Cannot write storage distribution. Error: %v", err)
	}
	err = ctx.WriteDistribution(DictionaryContextDir+"value-distribution.dat", ctx.ValueDictionary.frequency)
	if err != nil {
		log.Fatalf("Cannot write value distribution. Error: %v", err)
	}
}

// WriteDistribution writes distribution of operations and frequencies into given file
func (ctx *DictionaryContext) WriteDistribution(filename string, frequency []uint64) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Cannot open storage-dictionary file. Error: %v", err)
	}

	var total uint64 = 0
	for _, f := range frequency {
		total += f
	}

	s := len(frequency)

	frequencySorted := sortByFrequencyAcending(frequency)
	for _, fi := range frequencySorted {
		fmt.Fprintf(file, "%f - %f \n", float64(fi.opId)/float64(s), float64(fi.frequency)/float64(total))
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("Cannot close storage-dictionary file. Error: %v", err)
	}
	return nil
}

// FrequenciesWriter writes frequencies from dictionary recording
func (ctx *DictionaryContext) FrequenciesWriter() {
	file, err := os.OpenFile(DictionaryContextDir+"frequencies.dat", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Cannot open trace file. Error: %v", err)
	}
	fmt.Fprintln(file, "operations: ", ctx.OpFreq)
	fmt.Fprintln(file, "contractFreq: ", ctx.ContractFreq)
	fmt.Fprintln(file, "storageFreq: ", ctx.StorageFreq)
	fmt.Fprintln(file, "valueFreq: ", ctx.ValueFreq)

	if err := file.Close(); err != nil {
		log.Fatalf("Cannot close frequencies file. Error: %v", err)
	}
}

// sortByFrequencyAcending converts frequency slice with operation ids as indexes to structure,
// then sorts them in ascending order according to their frequencies
func sortByFrequencyAcending(frequency []uint64) []operationFrequency {
	var arr []operationFrequency
	for i, f := range frequency {
		arr = append(arr, operationFrequency{i, f})
	}

	sort.Slice(arr[:], func(i, j int) bool {
		return arr[i].frequency < arr[j].frequency
	})

	return arr
}
