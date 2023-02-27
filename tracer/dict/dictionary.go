package dict

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/dsnet/compress/bzip2"
)

// DictionaryLimit sets size of value dictionary.
var DictionaryLimit int = math.MaxInt - 1

// Dictionary data structure encodes/decodes a value
// to an index or vice versa.
type Dictionary[K comparable] struct {
	valueToIdx map[K]int // value to index map for encoding
	idxToValue []K       // value array for decoding
}

// Init initializes or clears a value dictionary.
func (d *Dictionary[K]) Init() {
	d.valueToIdx = map[K]int{}
	d.idxToValue = []K{}
}

// NewDictionary creates a new value dictionary.
func NewDictionary[K comparable]() *Dictionary[K] {
	p := new(Dictionary[K])
	p.Init()
	return p
}

// Encode a value to an index.
func (d *Dictionary[K]) Encode(value K) (int, error) {
	// find value
	idx, ok := d.valueToIdx[value]
	if !ok {
		idx = len(d.idxToValue)
		if idx >= DictionaryLimit {
			return 0, fmt.Errorf("Value dictionary exhausted")
		}
		d.valueToIdx[value] = idx
		d.idxToValue = append(d.idxToValue, value)
	}
	return idx, nil
}

// Decode an index to a value.
func (d *Dictionary[K]) Decode(idx int) (K, error) {
	if idx >= 0 && idx < len(d.idxToValue) {
		return d.idxToValue[idx], nil
	} else {
		var zeroValue K
		return zeroValue, fmt.Errorf("Invalid index")
	}
}

// Write dictionary to a binary file.
func (d *Dictionary[K]) Write(filename string) error {
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
func (d *Dictionary[K]) Read(filename string) error {
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
	var data K
	for ctr := 0; true; ctr++ {
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
