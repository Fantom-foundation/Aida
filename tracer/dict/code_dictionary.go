package dict

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// CodeDictionaryLimit sets the size of the code dictionary.
var CodeDictionaryLimit uint32 = math.MaxUint32 - 1

// CodeDictionary data structure encodes/decodes byte-code to an index or vice versa.
type CodeDictionary struct {
	codeToIdx map[string]uint32 // code (as string) to an index map for encoding
	idxToCode []string          // code  slice for decoding
}

// Init initializes or clears a code dictionary.
func (d *CodeDictionary) Init() {
	d.codeToIdx = map[string]uint32{}
	d.idxToCode = []string{}
}

// NewCodeDictionary creates a new code dictionary.
func NewCodeDictionary() *CodeDictionary {
	p := new(CodeDictionary)
	p.Init()
	return p
}

// Encode byte-code to a dictionary index.
func (d *CodeDictionary) Encode(code []byte) (uint32, error) {
	// find byte code
	sCode := string(code)
	idx, ok := d.codeToIdx[string(sCode)]
	if !ok {
		idx = uint32(len(d.idxToCode))
		if idx >= CodeDictionaryLimit {
			return 0, fmt.Errorf("Code dictionary exhausted")
		}
		d.codeToIdx[sCode] = idx
		d.idxToCode = append(d.idxToCode, sCode)
	}
	return idx, nil
}

// Decode a dictionary index to byte-code.
func (d *CodeDictionary) Decode(idx uint32) ([]byte, error) {
	if idx < uint32(len(d.idxToCode)) {
		return []byte(d.idxToCode[idx]), nil
	} else {
		return nil, fmt.Errorf("Index out-of-bound")
	}
}

// Write dictionary to a binary file.
func (d *CodeDictionary) Write(filename string) error {
	// open code dictionary file for writing
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// write all dictionary entries
	for _, code := range d.idxToCode {

		// write length of code block
		if len(code) >= math.MaxUint32 {
			return fmt.Errorf("Code is too large to write")
		}
		length := uint32(len(code))
		err := binary.Write(f, binary.LittleEndian, &length)
		if err != nil {
			return fmt.Errorf("Error writing code length. Error: %v", err)
		}

		// write code
		if _, err := f.Write([]byte(code)); err != nil {
			return fmt.Errorf("Error writing byte-code. Error: %v", err)
		}
	}
	return nil
}

// Read dictionary from a binary file.
func (d *CodeDictionary) Read(filename string) error {
	// clear code dictionary
	d.Init()

	// open code dictionary file for reading
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// read entries from file
	for ctr := uint32(0); true; ctr++ {
		// read next entry

		// read length
		var length uint32
		err := binary.Read(f, binary.LittleEndian, &length)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("Code dictionary file/reading is corrupted. Error: %v", err)
		}

		// read byte code
		code := make([]byte, length)
		n, err := f.Read(code)
		if err != nil {
			return fmt.Errorf("Error reading code length/file is corrupted. Error: %v", err)
		} else if n != int(length) {
			return fmt.Errorf("Error reading code length/wrong size")
		}

		// encode entry
		idx, err := d.Encode(code)
		if err != nil {
			return fmt.Errorf("Failed to encode byte-code while reading. Error: %v", err)
		} else if idx != ctr {
			return fmt.Errorf("Corrupted code dictionary file entries")
		}
	}
	return nil
}
