package dict

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
)

// Entry limit of code dictionary
var CodeDictionaryLimit uint32 = math.MaxUint32 - 1

// Dictionary data structure encodes/decodes byte code to a
// dictionary index or vice versa.
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

// Encode byte code to a dictionary index.
func (d *CodeDictionary) Encode(code []byte) (uint32, error) {
	// find byte code
	sCode := string(code)
	idx, ok := d.codeToIdx[string(sCode)]
	if !ok {
		idx = uint32(len(d.idxToCode))
		if idx >= CodeDictionaryLimit {
			return 0, errors.New("Code dictionary exhausted")
		}
		d.codeToIdx[sCode] = idx
		d.idxToCode = append(d.idxToCode, sCode)
	}
	return idx, nil
}

// Decode a dictionary index to byte code.
func (d *CodeDictionary) Decode(idx uint32) ([]byte, error) {
	if idx < uint32(len(d.idxToCode)) {
		return []byte(d.idxToCode[idx]), nil
	} else {
		return nil, errors.New("Index out-of-bound")
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
			return errors.New("Code size exceeds uint32")
		}
		length := uint32(len(code))
		err := binary.Write(f, binary.LittleEndian, &length)
		if err != nil {
			return err
		}

		// write code
		if _, err := f.Write([]byte(code)); err != nil {
			return err
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
			return errors.New("Code dictionary file/reading is corrupted")
		}

		// read byte code
		code := make([]byte, length)
		n, err := f.Read(code)
		if err != nil {
			return errors.New("Code dictionary file/reading is corrupted")
		} else if n != int(length) {
			return errors.New("Code byte has wrong file size.")
		}

		// encode entry
		idx, err := d.Encode(code)

		if err != nil {
			return err
		} else if idx != ctr {
			return errors.New("Corrupted code dictionary file entries")
		}
	}
	return nil
}
