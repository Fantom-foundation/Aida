package context

import (
	"bufio"
	"log"
	"os"

	"github.com/dsnet/compress/bzip2"
	"github.com/ethereum/go-ethereum/common"
)

const (
	WriteBufferSize = 1048576 // Size of write buffer for writing trace file.
)

// Context is an environment/facade for recording and replaying trace files
type Context struct {
	prevContract common.Address // previously used contract
	keyCache     *KeyCache      // key cache
}

// Record is the recording environment/facade
type Record struct {
	Context
	Debug bool          // debug flag
	file  *os.File      // trace file
	bFile *bufio.Writer // buffer for trace file
	ZFile *bzip2.Writer // compressed file
}

// Replay is the replaying environment/facade
type Replay struct {
	Context
	snapshot *SnapshotIndex // snapshot translation table for replay
}

// NewContext creates a new replay context.
func NewReplay() *Replay {
	return &Replay{
		Context: Context{prevContract: common.Address{},
			keyCache: NewKeyCache()},
		snapshot: NewSnapshotIndex(),
	}
}

// NewContext creates a new record context.
func NewRecord(filename string) *Record {
	// open trace file, write buffer, and compressed stream
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0744)
	if err != nil {
		log.Fatalf("Cannot open trace file. Error: %v", err)
	}
	bFile := bufio.NewWriterSize(file, WriteBufferSize)
	ZFile, err := bzip2.NewWriter(bFile, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		log.Fatalf("Cannot open bzip2 stream. Error: %v", err)
	}
	return &Record{
		Context: Context{prevContract: common.Address{},
			keyCache: NewKeyCache()},
		file:  file,
		bFile: bFile,
		ZFile: ZFile,
	}
}

// Close the trace file in the record context.
func (ctx *Record) Close() {
	// closing compressed stream, flushing buffer, and closing trace file
	if err := ctx.ZFile.Close(); err != nil {
		log.Fatalf("Cannot close bzip2 writer. Error: %v", err)
	}
	if err := ctx.bFile.Flush(); err != nil {
		log.Fatalf("Cannot flush buffer. Error: %v", err)
	}
	if err := ctx.file.Close(); err != nil {
		log.Fatalf("Cannot close trace file. Error: %v", err)
	}
}

////////////////////////////////////////////////////////////////
// Contract methods
////////////////////////////////////////////////////////////////

// EncodeContract encodes a given contract address and returns contract's address.
func (ctx *Context) EncodeContract(contract common.Address) common.Address {
	ctx.prevContract = contract
	return contract
}

// DecodeContract decodes the contract address.
func (ctx *Context) DecodeContract(contract common.Address) common.Address {
	ctx.prevContract = contract
	return contract
}

// PrevContract returns the previously used contract address.
func (ctx *Context) PrevContract() common.Address {
	return ctx.prevContract
}

////////////////////////////////////////////////////////////////
// Storage methods
////////////////////////////////////////////////////////////////

// EncodeKey encodes a storage key and returns an index and the key.
func (ctx *Context) EncodeKey(key common.Hash) (common.Hash, int) {
	pos := ctx.keyCache.Place(key)
	return key, pos
}

// DecodeKey decodes a storage address.
func (ctx *Context) DecodeKey(key common.Hash) common.Hash {
	ctx.keyCache.Place(key)
	return key
}

// DecodeKeyCache reads from cache with updating index cache.
func (ctx *Context) DecodeKeyCache(sPos int) common.Hash {
	key, err := ctx.keyCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be looked up. Error: %v", err)
	}
	return ctx.DecodeKey(key)
}

// ReadKeyCache reads from cache without updating index cache.
func (ctx *Context) ReadKeyCache(sPos int) common.Hash {
	key, err := ctx.keyCache.Get(sPos)
	if err != nil {
		log.Fatalf("Storage position could not be found. Error: %v", err)
	}
	return key
}

////////////////////////////////////////////////////////////////
// Snapshot methods
////////////////////////////////////////////////////////////////

// InitSnapshot initializes snapshot map.
func (ctx *Replay) InitSnapshot() {
	ctx.snapshot.Init()
}

// AddSnapshot adds map between recorded/replayed snapshot-id.
func (ctx *Replay) AddSnapshot(recordedID int32, replayedID int32) {
	ctx.snapshot.Add(recordedID, replayedID)
}

// GetSnapshot gets snapshot-id.
func (ctx *Replay) GetSnapshot(recordedID int32) int32 {
	replayedID, err := ctx.snapshot.Get(recordedID)
	if err != nil {
		log.Fatalf("Replayed Snapshot ID is missing. Error: %v", err)
	}
	return replayedID
}
