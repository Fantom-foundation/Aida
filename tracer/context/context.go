// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package context

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/profile"
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
	Profile  bool           // if true collect stats
	Stats    *profile.Stats // stats object
}

// NewReplay creates a new replay context.
func NewReplay() *Replay {
	return &Replay{
		Context: Context{prevContract: common.Address{},
			keyCache: NewKeyCache()},
		snapshot: NewSnapshotIndex(),
	}
}

func (ctx *Replay) EnableProfiling(csv string) {
	ctx.Profile = true
	ctx.Stats = profile.NewStats(csv)
}

// NewContext creates a new record context.
func NewRecord(filename string, first uint64) (*Record, error) {
	// open trace file, write buffer, and compressed stream
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return nil, fmt.Errorf("cannot open trace file; %v", err)
	}
	bFile := bufio.NewWriterSize(file, WriteBufferSize)
	ZFile, err := bzip2.NewWriter(bFile, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		return nil, fmt.Errorf("cannot open bzip2 stream; %v", err)
	}
	// write header
	if err := binary.Write(ZFile, binary.LittleEndian, first); err != nil {
		return nil, fmt.Errorf("fail to write file header")
	}

	return &Record{
		Context: Context{prevContract: common.Address{},
			keyCache: NewKeyCache()},
		file:  file,
		bFile: bFile,
		ZFile: ZFile,
	}, nil
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
