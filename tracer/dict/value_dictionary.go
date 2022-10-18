package dict

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math"
	"os"
)

// Entry limit of storage dictionary
var ValueDictionaryLimit uint64 = math.MaxUint64

// Dictionary data structure encodes/decodes a storage address
// to a dictionary index or vice versa.
type ValueDictionary struct {
	storageToIdx map[common.Hash]uint64 // storage address to index map for encoding
	idxToValue   []common.Hash          // storage address slice for decoding
}

// Init initializes or clears a storage dictionary.
func (sDict *ValueDictionary) Init() {
	sDict.storageToIdx = map[common.Hash]uint64{}
	sDict.idxToValue = []common.Hash{}
}

// Create a new storage dictionary.
func NewValueDictionary() *ValueDictionary {
	p := new(ValueDictionary)
	p.Init()
	return p
}

// Encode an storage address to a dictionary index.
func (sDict *ValueDictionary) Encode(addr common.Hash) (uint64, error) {
	// find storage address
	idx, ok := sDict.storageToIdx[addr]
	if !ok {
		idx = uint64(len(sDict.idxToValue))
		if idx >= ValueDictionaryLimit {
			return 0, errors.New("Value dictionary exhausted")
		}
		sDict.storageToIdx[addr] = idx
		sDict.idxToValue = append(sDict.idxToValue, addr)
	}
	return idx, nil
}

// Decode a dictionary index to an address.
func (sDict *ValueDictionary) Decode(idx uint64) (common.Hash, error) {
	if idx < uint64(len(sDict.idxToValue)) {
		return sDict.idxToValue[idx], nil
	} else {
		return common.Hash{}, errors.New("Index out-of-bound")
	}
}

// Write dictionary to a binary file.
func (sDict *ValueDictionary) Write(filename string) error {
	// open storage dictionary file for writing
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// write all dictionary entries
	for _, addr := range sDict.idxToValue {
		data := addr.Bytes()
		if _, err := f.Write(data); err != nil {
			return err
		}
	}
	return nil
}

// Read dictionary from a binary file.
func (sDict *ValueDictionary) Read(filename string) error {
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
	for ctr := uint64(0); true; ctr++ {
		// read next entry
		n, err := f.Read(data)
		if n == 0 {
			break
		} else if n < len(data) || err != nil {
			return errors.New("Value dictionary file/reading is corrupted")
		}

		// encode entry
		idx, err := sDict.Encode(common.BytesToHash(data))
		if err != nil {
			return err
		} else if idx != ctr {
			return errors.New("Corrupted storage dictionary file entries")
		}
	}
	return nil
}
