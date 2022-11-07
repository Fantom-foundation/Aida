package dict

import (
	"fmt"
	"math"
	"os"

	"github.com/dsnet/compress/bzip2"
	"github.com/ethereum/go-ethereum/common"
)

// ContractDictionaryLimit sets the size of the contract dictionary.
var ContractDictionaryLimit uint32 = math.MaxUint32 - 1

// ContractDictionary data structure encodes/decodes a contract address to an index or vice versa.
type ContractDictionary struct {
	contractToIdx map[common.Address]uint32 // contract address to index map for encoding
	idxToContract []common.Address          // contract address slice for decoding
}

// Init initializes or clears a contract dictionary.
func (cDict *ContractDictionary) Init() {
	cDict.contractToIdx = map[common.Address]uint32{}
	cDict.idxToContract = []common.Address{}
}

// NewContractDictionary creates a new contract dictionary.
func NewContractDictionary() *ContractDictionary {
	p := new(ContractDictionary)
	p.Init()
	return p
}

// Encode an contract address to a dictionary index.
func (cDict *ContractDictionary) Encode(addr common.Address) (uint32, error) {
	// find contract address
	idx, ok := cDict.contractToIdx[addr]
	if !ok {
		idx = uint32(len(cDict.idxToContract))
		if idx >= ContractDictionaryLimit {
			return 0, fmt.Errorf("Contract dictionary exhausted")
		}
		cDict.contractToIdx[addr] = idx
		cDict.idxToContract = append(cDict.idxToContract, addr)
	}
	return idx, nil
}

// Decode a dictionary index to an address.
func (cDict *ContractDictionary) Decode(idx uint32) (common.Address, error) {
	if idx < uint32(len(cDict.idxToContract)) {
		return cDict.idxToContract[idx], nil
	} else {
		return common.Address{}, fmt.Errorf("Index out-of-bound")
	}
}

// Write dictionary to a binary file.
func (cDict *ContractDictionary) Write(filename string) error {
	// open contract dictionary file for writing
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Cannot open contract-dictionary file. Error: %v", err)
	}
	zfile, err := bzip2.NewWriter(file, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		return fmt.Errorf("Cannot open bzip2 stream of contract-dictionary. Error: %v", err)
	}
	// write all dictionary entries
	for _, addr := range cDict.idxToContract {
		data := addr.Bytes()
		if _, err := zfile.Write(data); err != nil {
			return err
		}
	}
	// close bzip2 stream and file
	if err := zfile.Close(); err != nil {
		return fmt.Errorf("Cannot close bzip2 stream of contract-dictionary. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close contract-dictionary file. Error: %v", err)
	}
	return nil
}

// Read dictionary from a binary file.
func (cDict *ContractDictionary) Read(filename string) error {
	// clear contract dictionary
	cDict.Init()
	// open code dictionary file for reading, read buffer, and gzip stream
	file, err1 := os.Open(filename)
	if err1 != nil {
		return fmt.Errorf("Cannot open contract-dictionary file. Error: %v", err1)
	}
	zfile, err2 := bzip2.NewReader(file, &bzip2.ReaderConfig{})
	if err2 != nil {
		return fmt.Errorf("Cannot open bzip2 stream of contract-dictionary. Error: %v", err2)
	}
	// read entries from file
	data := common.Address{}.Bytes()
	for ctr := uint32(0); true; ctr++ {
		// read next entry
		n, err := zfile.Read(data)
		if n == 0 {
			break
		} else if n < len(data) {
			return fmt.Errorf("Error reading address/wrong size")
		} else if err != nil {
			return fmt.Errorf("Error reading address. Error:%v", err)
		}
		// encode entry
		idx, err := cDict.Encode(common.BytesToAddress(data))
		if err != nil {
			return fmt.Errorf("Error encoding address. Error:%v", err)
		} else if idx != ctr {
			return fmt.Errorf("Corrupted contract dictionary file entries")
		}
	}
	// close file
	if err := zfile.Close(); err != nil {
		return fmt.Errorf("Cannot close bzip2 stream of contract-dictionary. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close contract-dictionary file. Error: %v", err)
	}
	return nil
}
