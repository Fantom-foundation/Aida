package tracer

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// BlockIndex maps a block number to a file position.
// The file position points to the first state operation of the block.
type BlockIndex struct {
	blockToFPos map[uint64]int64 // block number -> file position
}

// Initialize a block-index.
func (oIdx *BlockIndex) Init() {
	oIdx.blockToFPos = make(map[uint64]int64)
}

// Create new block-index data structure.
func NewBlockIndex() *BlockIndex {
	p := new(BlockIndex)
	p.Init()
	return p
}

// Add new entry file-position for a block.
func (oIdx *BlockIndex) Add(block uint64, fpos int64) error {
	var err error = nil
	if _, ok := oIdx.blockToFPos[block]; ok {
		err = errors.New("block number already exists")
	}
	oIdx.blockToFPos[block] = fpos
	return err
}

// Check whether block-index has a file position for a block number.
func (oIdx *BlockIndex) Exists(block uint64) (bool, error) {
	operation, ok := oIdx.blockToFPos[block]
	return ok, nil
}

// Obtain file position number for a block number.
func (oIdx *BlockIndex) Get(block uint64) (int64, error) {
	operation, ok := oIdx.blockToFPos[block]
	if !ok {
		return 0, errors.New("block number does not exist")
	}
	return operation, nil
}

// Write the block-index to a file.
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
		writeSlice(data)
	}
	return nil
}

// Read block-index from a file.
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
		} else if err != nil {
			return err
		}
		err = oIdx.Add(data.Block, data.FPos)
		if err != nil {
			return err
		}
	}
	return nil
}
