package tracer

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// BlockIndex maps a block number to a file position where its first state operation is stored.
type BlockIndex struct {
	blockToFPos map[uint64]int64 // block number -> file position
}

// Initialize an operation index.
func (oIdx *BlockIndex) Init() {
	oIdx.blockToFPos = make(map[uint64]int64)
}

// Create new BlockIndex data structure.
func NewBlockIndex() *BlockIndex {
	p := new(BlockIndex)
	p.Init()
	return p
}

// Add new entry.
func (oIdx *BlockIndex) Add(block uint64, fpos int64) error {
	var err error = nil
	if _, ok := oIdx.blockToFPos[block]; ok {
		err = errors.New("block number already exists")
	}
	oIdx.blockToFPos[block] = fpos
	return err
}

// Get operation number.
func (oIdx *BlockIndex) Get(block uint64) (int64, error) {
	operation, ok := oIdx.blockToFPos[block]
	if !ok {
		return 0, errors.New("block number does not exist")
	}
	return operation, nil
}

// Write index to a binary file.
func (oIdx *BlockIndex) Write(filename string) error {
	// open index file for writing
	f, err := os.OpenFile(TraceDir+filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// write all dictionary entries
	for block, fpos := range oIdx.blockToFPos {
		var data = []any{block, fpos}
		for _, value := range data {
			err := binary.Write(f, binary.LittleEndian, value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Read dictionary from a binary file.
func (oIdx *BlockIndex) Read(filename string) error {
	// clear storage dictionary
	oIdx.Init()

	// open storage dictionary file for reading
	f, err := os.OpenFile(TraceDir+filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
	}()

	// read entries from file
	for {
		// read next entry
		var data struct {
			Block uint64
			FPos  int64
		}
		err := binary.Read(f, binary.LittleEndian, &data)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = oIdx.Add(data.Block, data.FPos)
		if err != nil {
			return err
		}
	}
	return nil
}
