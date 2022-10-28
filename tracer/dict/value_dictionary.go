package dict

import (
	"github.com/ethereum/go-ethereum/common"
	"math"
	"os"
	"fmt"
)

// ValueDictionaryLimit sets size of storage dictionary.
var ValueDictionaryLimit uint64 = math.MaxUint64 - 1

// ValueDictionary data structure encodes/decodes a value 
// to an index or vice versa.
type ValueDictionary struct {
	storageToIdx map[common.Hash]uint64 // value to index map for encoding
	idxToValue   []common.Hash          // value array for decoding
}

// Init initializes or clears a value dictionary.
func (sDict *ValueDictionary) Init() {
	sDict.storageToIdx = map[common.Hash]uint64{}
	sDict.idxToValue = []common.Hash{}
}

// NewValueDictionary creates a new value dictionary.
func NewValueDictionary() *ValueDictionary {
	p := new(ValueDictionary)
	p.Init()
	return p
}

// Encode a value to an index.
func (sDict *ValueDictionary) Encode(addr common.Hash) (uint64, error) {
	// find storage address
	idx, ok := sDict.storageToIdx[addr]
	if !ok {
		idx = uint64(len(sDict.idxToValue))
		if idx >= ValueDictionaryLimit {
			return 0, fmt.Errorf("Value dictionary exhausted")
		}
		sDict.storageToIdx[addr] = idx
		sDict.idxToValue = append(sDict.idxToValue, addr)
	}
	return idx, nil
}

// Decode an index to a value.
func (sDict *ValueDictionary) Decode(idx uint64) (common.Hash, error) {
	if idx < uint64(len(sDict.idxToValue)) {
		return sDict.idxToValue[idx], nil
	} else {
		return common.Hash{}, fmt.Errorf("Index out-of-bound")
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
		} else if n < len(data) {
			return fmt.Errorf("Error reading value/wrong size")
		} else if err != nil {
			return fmt.Errorf("Error reading value. Error: %v", err)
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
