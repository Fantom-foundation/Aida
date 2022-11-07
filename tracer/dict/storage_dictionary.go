package dict

import (
	"fmt"
	"math"
	"os"

	"github.com/dsnet/compress/bzip2"
	"github.com/ethereum/go-ethereum/common"
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
func (d *StorageDictionary) Init() {
	d.storageToIdx = map[common.Hash]uint32{}
	d.idxToStorage = []common.Hash{}
}

// NewStorageDictionary creates a new storage dictionary.
func NewStorageDictionary() *StorageDictionary {
	p := new(StorageDictionary)
	p.Init()
	return p
}

// Encode an storage address to an index.
func (d *StorageDictionary) Encode(addr common.Hash) (uint32, error) {
	// find storage address
	idx, ok := d.storageToIdx[addr]
	if !ok {
		idx = uint32(len(d.idxToStorage))
		if idx >= StorageDictionaryLimit {
			return 0, fmt.Errorf("Storage dictionary exhausted")
		}
		d.storageToIdx[addr] = idx
		d.idxToStorage = append(d.idxToStorage, addr)
	}
	return idx, nil
}

// Decode a dictionary index to an address.
func (d *StorageDictionary) Decode(idx uint32) (common.Hash, error) {
	if idx < uint32(len(d.idxToStorage)) {
		return d.idxToStorage[idx], nil
	} else {
		return common.Hash{}, fmt.Errorf("Index out-of-bound")
	}
}

// Write dictionary to a binary file.
func (d *StorageDictionary) Write(filename string) error {
	// open storage dictionary file for writing
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Cannot open storage-dictionary file. Error: %v", err)
	}
	zfile, err := bzip2.NewWriter(file, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		return fmt.Errorf("Cannot open bzip2 stream for storage-dictionary. Error: %v", err)
	}
	// write all dictionary entries
	for _, addr := range d.idxToStorage {
		data := addr.Bytes()
		if _, err := zfile.Write(data); err != nil {
			return err
		}
	}
	// close file
	if err := zfile.Close(); err != nil {
		return fmt.Errorf("Cannot close bzip2 stream of storage-dictionary. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close storage-dictionary file. Error: %v", err)
	}
	return nil
}

// Read dictionary from a binary file.
func (d *StorageDictionary) Read(filename string) error {
	// clear storage dictionary
	d.Init()
	// open code dictionary file for reading, read buffer, and gzip stream
	file, err1 := os.Open(filename)
	if err1 != nil {
		return fmt.Errorf("Cannot open storage-dictionary file. Error: %v", err1)
	}
	zfile, err2 := bzip2.NewReader(file, &bzip2.ReaderConfig{})
	if err2 != nil {
		return fmt.Errorf("Cannot open bzip2 stream of storage-dictionary. Error: %v", err2)
	}
	// read entries from file
	data := common.Hash{}.Bytes()
	for ctr := uint32(0); true; ctr++ {
		// read next entry
		n, err := zfile.Read(data)
		if n == 0 {
			break
		} else if n < len(data) {
			return fmt.Errorf("Error reading storage address/wrong size.")
		} else if err != nil {
			return fmt.Errorf("Error reading storage address. Error: %v", err)
		}
		// encode entry
		idx, err := d.Encode(common.BytesToHash(data))
		if err != nil {
			return err
		} else if idx != ctr {
			return fmt.Errorf("Corrupted storage dictionary file entries")
		}
	}
	// close file
	if err := zfile.Close(); err != nil {
		return fmt.Errorf("Cannot close bzip2 stream of storage-dictionary. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close storage-dictionary file. Error: %v", err)
	}
	return nil
}
