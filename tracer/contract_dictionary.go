package tracer

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math"
	"os"
)

// Entry limit of contract dictionary
var ContractDictionaryLimit uint32 = math.MaxUint32

// Dictionary data structure encodes/decodes a contract address
// to a dictionary index or vice versa.
type ContractDictionary struct {
	contractToIdx map[common.Address]uint32 // contract address to index map for encoding
	idxToContract []common.Address          // contract address slice for decoding
}

// Init initializes or clears a contract dictionary.
func (cDict *ContractDictionary) Init() {
	cDict.contractToIdx = map[common.Address]uint32{}
	cDict.idxToContract = []common.Address{}
}

// Create a new contract dictionary.
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
			return 0, errors.New("Contract dictionary exhausted")
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
		return common.Address{}, errors.New("Index out-of-bound")
	}
}

// Write dictionary to a binary file.
func (cDict *ContractDictionary) Write(filename string) error {
	// open contract dictionary file for writing
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// write all dictionary entries
	for _, addr := range cDict.idxToContract {
		data := addr.Bytes()
		if _, err := f.Write(data); err != nil {
			return err
		}
	}
	return nil
}

// Read dictionary from a binary file.
func (cDict *ContractDictionary) Read(filename string) error {
	// clear contract dictionary
	cDict.Init()

	// open contract dictionary file for reading
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// read entries from file
	data := common.Address{}.Bytes()
	for ctr := uint32(0); true; ctr++ {
		// read next entry
		n, err := f.Read(data)
		if n == 0 {
			break
		} else if n < len(data) || err != nil {
			return errors.New("Contract dictionary file/reading is corrupted")
		}

		// encode entry
		idx, err := cDict.Encode(common.BytesToAddress(data))
		if err != nil {
			return err
		} else if idx != ctr {
			return errors.New("Corrupted contract dictionary file entries")
		}
	}
	return nil
}
