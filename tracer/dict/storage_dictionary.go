package dict

import (
	"github.com/ethereum/go-ethereum/common"
	"math"
	"os"
	"fmt"
)

// StorageDictionaryLimit sets the size of storage dictionary.
var StorageDictionaryLimit uint32 = math.MaxUint32 - 1

// Dictionary data structure encodes/decodes a storage address
// to a dictionary index or vice versa.
type StorageDictionary struct {
	storageToIdx map[common.Hash]uint32 // storage address to index map for encoding
	idxToStorage []common.Hash          // storage address slice for decoding
}

// Init initializes or clears a storage dictionary.
func (sDict *StorageDictionary) Init() {
	sDict.storageToIdx = map[common.Hash]uint32{}
	sDict.idxToStorage = []common.Hash{}
}

// NewStorageDictionary creates a new storage dictionary.
func NewStorageDictionary() *StorageDictionary {
	p := new(StorageDictionary)
	p.Init()
	return p
}

// Encode an storage address to an index.
func (sDict *StorageDictionary) Encode(addr common.Hash) (uint32, error) {
	// find storage address
	idx, ok := sDict.storageToIdx[addr]
	if !ok {
		idx = uint32(len(sDict.idxToStorage))
		if idx >= StorageDictionaryLimit {
			return 0, fmt.Errorf("Storage dictionary exhausted")
		}
		sDict.storageToIdx[addr] = idx
		sDict.idxToStorage = append(sDict.idxToStorage, addr)
	}
	return idx, nil
}

// Decode a dictionary index to an address.
func (sDict *StorageDictionary) Decode(idx uint32) (common.Hash, error) {
	if idx < uint32(len(sDict.idxToStorage)) {
		return sDict.idxToStorage[idx], nil
	} else {
		return common.Hash{}, fmt.Errorf("Index out-of-bound")
	}
}

// Write dictionary to a binary file.
func (sDict *StorageDictionary) Write(filename string) error {
	// open storage dictionary file for writing
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// write all dictionary entries
	for _, addr := range sDict.idxToStorage {
		data := addr.Bytes()
		if _, err := f.Write(data); err != nil {
			return err
		}
	}
	return nil
}

// Read dictionary from a binary file.
func (sDict *StorageDictionary) Read(filename string) error {
	// clear storage dictionary
	sDict.Init()

	// open storage dictionary file for reading
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// read entries from file
	data := common.Hash{}.Bytes()
	for ctr := uint32(0); true; ctr++ {
		// read next entry
		n, err := f.Read(data)
		if n == 0 {
			break
		} else if n < len(data) {
			return fmt.Errorf("Error reading storage address/wrong size.")
		} else if err != nil {
			return fmt.Errorf("Error reading storage address. Error: %v", err)
		}

		// encode entry
		idx, err := sDict.Encode(common.BytesToHash(data))
		if err != nil {
			return err
		} else if idx != ctr {
			return fmt.Errorf("Corrupted storage dictionary file entries")
		}
	}
	return nil
}
