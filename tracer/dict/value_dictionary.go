package dict

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/dsnet/compress/bzip2"
	"github.com/ethereum/go-ethereum/common"
)

// ValueDictionaryLimit sets size of value dictionary.
var ValueDictionaryLimit uint64 = math.MaxUint64 - 1

// ValueDictionary data structure encodes/decodes a value
// to an index or vice versa.
type ValueDictionary struct {
	valueToIdx map[common.Hash]uint64 // value to index map for encoding
	idxToValue []common.Hash          // value array for decoding
}

// Init initializes or clears a value dictionary.
func (d *ValueDictionary) Init() {
	d.valueToIdx = map[common.Hash]uint64{}
	d.idxToValue = []common.Hash{}
}

// NewValueDictionary creates a new value dictionary.
func NewValueDictionary() *ValueDictionary {
	p := new(ValueDictionary)
	p.Init()
	return p
}

// Encode a value to an index.
func (d *ValueDictionary) Encode(value common.Hash) (uint64, error) {
	// find value
	idx, ok := d.valueToIdx[value]
	if !ok {
		idx = uint64(len(d.idxToValue))
		if idx >= ValueDictionaryLimit {
			return 0, fmt.Errorf("Value dictionary exhausted")
		}
		d.valueToIdx[value] = idx
		d.idxToValue = append(d.idxToValue, value)
	}
	return idx, nil
}

// Decode an index to a value.
func (d *ValueDictionary) Decode(idx uint64) (common.Hash, error) {
	if idx < uint64(len(d.idxToValue)) {
		return d.idxToValue[idx], nil
	} else {
		return common.Hash{}, fmt.Errorf("Index out-of-bound")
	}
}

// Write dictionary to a binary file.
func (d *ValueDictionary) Write(filename string) error {
	// open code dictionary file for writing
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Cannot open value-dictionary file. Error:%v", err)
	}
	zfile, err := bzip2.NewWriter(file, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		return fmt.Errorf("Cannot open bzip2 stream of value-dictionary. Error: %v", err)
	}
	// write magic number
	magic := uint64(4714)
	if err := binary.Write(zfile, binary.LittleEndian, &magic); err != nil {
		return fmt.Errorf("Error writing magic number. Error: %v", err)
	}
	// write all dictionary entries
	for _, value := range d.idxToValue {
		if err := binary.Write(zfile, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	// close file
	if err := zfile.Close(); err != nil {
		return fmt.Errorf("Cannot close bzip2 stream of value-dictionary. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close value-dictionary file. Error: %v", err)
	}
	return nil
}

// Read dictionary from a binary file.
func (d *ValueDictionary) Read(filename string) error {
	// clear value dictionary
	d.Init()
	// open code dictionary file for reading, read buffer, and bzip2 stream
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("Cannot open value-dictionary file. Error:%v", err)
	}
	zfile, err := bzip2.NewReader(file, &bzip2.ReaderConfig{})
	if err != nil {
		return fmt.Errorf("Cannot open bzip stream of value-dictionary. Error: %v", err)
	}
	// read and check magic number
	var magic uint64
	if err := binary.Read(zfile, binary.LittleEndian, &magic); err != nil && magic != uint64(4714) {
		return fmt.Errorf("Cannot read magic number; code-dictionary is corrupted. Error: %v", err)
	}
	// read entries from file
	data := common.Hash{}
	for ctr := uint64(0); true; ctr++ {
		// read next entry
		if err := binary.Read(zfile, binary.LittleEndian, &data); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("Error reading storage address. Error: %v", err)
		}
		// encode entry
		idx, err := d.Encode(data)
		if err != nil {
			return err
		} else if idx != ctr {
			return fmt.Errorf("Corrupted value dictionary file entries")
		}
	}
	// close file
	if err := zfile.Close(); err != nil {
		return fmt.Errorf("Cannot close bzip2 stream. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close trace file. Error: %v", err)
	}
	return nil
}
