package dictionary

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/dsnet/compress/bzip2"
)

// DictionaryLimit sets size of dictionary.
var DictionaryLimit int = math.MaxInt - 1

// Dictionary data structure encodes/decodes a value to an index or vice versa.
type Dictionary[K comparable] struct {
	valueToIdx map[K]int // value to index map for encoding
	idxToValue []K       // value array for decoding
}

// Init initializes or clears a dictionary.
func (d *Dictionary[K]) Init() {
	d.valueToIdx = map[K]int{}
	d.idxToValue = []K{}
}

// NewDictionary creates a new dictionary.
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
			return 0, fmt.Errorf("dictionary exhausted")
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

// Write dictionary to a binary file. Keys have fixed length.
func (d *Dictionary[K]) Write(filename string, magic uint64) error {
	// open dictionary file for writing
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Cannot open dictionary file. Error:%v", err)
	}
	zfile, err := bzip2.NewWriter(file, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		return fmt.Errorf("Cannot open bzip2 stream of dictionary. Error: %v", err)
	}
	// write magic number
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
		return fmt.Errorf("Cannot close bzip2 stream of dictionary. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close dictionary file. Error: %v", err)
	}
	return nil
}

// Read dictionary from a binary file. Keys have fixed length.
func (d *Dictionary[K]) Read(filename string, magic uint64) error {
	// clear dictionary
	d.Init()
	// open dictionary file for reading, read buffer, and bzip2 stream
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("Cannot open dictionary file. Error:%v", err)
	}
	zfile, err := bzip2.NewReader(file, &bzip2.ReaderConfig{})
	if err != nil {
		return fmt.Errorf("Cannot open bzip stream of dictionary. Error: %v", err)
	}
	// read and check magic number
	var magicData uint64
	if err := binary.Read(zfile, binary.LittleEndian, &magicData); err != nil && magic != magicData {
		return fmt.Errorf("Cannot read magic number; dictionary is corrupted. Error: %v", err)
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
			return fmt.Errorf("Corrupted dictionary file entries")
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

// Write dictionary to a binary file. Key is a string.
func (d *Dictionary[K]) WriteString(filename string, magic uint64) error {
	// open dictionary file for writing
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Cannot open dictionary file. Error: %v", err)
	}
	zfile, err := bzip2.NewWriter(file, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		return fmt.Errorf("Cannot open bzip2 stream for dictionary. Error: %v", err)
	}
	// write magic number
	if err := binary.Write(zfile, binary.LittleEndian, &magic); err != nil {
		return fmt.Errorf("Error writing magic number. Error: %v", err)
	}
	// write all dictionary entries
	for _, value := range d.idxToValue {
		// write length of value block
		str := any(value).(string)
		if len(str) >= math.MaxUint32 {
			return fmt.Errorf("string value is too large to write")
		}
		length := uint32(len(str))
		if err := binary.Write(zfile, binary.LittleEndian, length); err != nil {
			return fmt.Errorf("Error writing value length. Error: %v", err)
		}
		// write value
		if err := binary.Write(zfile, binary.LittleEndian, []byte(str)); err != nil {
			return fmt.Errorf("Error writing byte-value. Error: %v", err)
		}
	}
	// close file
	if err := zfile.Close(); err != nil {
		return fmt.Errorf("Cannot close bzip2 stream of dictionary. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close dictionary file. Error: %v", err)
	}
	return nil
}

// Read dictionary from a binary file. Key is a string.
func (d *Dictionary[K]) ReadString(filename string, magic uint64) error {
	// clear dictionary
	d.Init()
	// open dictionary file for reading, read buffer, and gzip stream
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("Cannot open dictionary file. Error: %v", err)
	}
	zfile, err := bzip2.NewReader(file, &bzip2.ReaderConfig{})
	if err != nil {
		return fmt.Errorf("Cannot open bzip2 stream of dictionary. Error: %v", err)
	}
	// read and check magic number
	var magicData uint64
	if err := binary.Read(zfile, binary.LittleEndian, &magicData); err != nil && magicData != magic {
		return fmt.Errorf("Cannot read magic number; dictionary is corrupted. Error: %v", err)
	}
	// read entries from file
	for ctr := 0; true; ctr++ {
		// read length of byte-value
		var length uint32
		err := binary.Read(zfile, binary.LittleEndian, &length)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("dictionary file/reading is corrupted. Error: %v", err)
		}
		// read byte-value
		value := make([]byte, length)
		if err := binary.Read(zfile, binary.LittleEndian, value); err != nil {
			return fmt.Errorf("Error reading value length/file is corrupted. Error: %v", err)
		}
		str := string(value)
		// encode byte-value entry
		idx, err := d.Encode(any(str).(K))
		if err != nil {
			return fmt.Errorf("Failed to encode byte-value while reading. Error: %v", err)
		} else if idx != ctr {
			return fmt.Errorf("Corrupted dictionary file entries")
		}
	}
	if err := zfile.Close(); err != nil {
		return fmt.Errorf("Cannot close bzip2 stream of dictionary. Error: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("Cannot close dictionary file. Error: %v", err)
	}
	return nil
}
