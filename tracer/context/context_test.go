// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package context

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestContextEncodeContract encodes an address and check the returned index.
func TestContextEncodeContract(t *testing.T) {
	ctx := NewReplay()
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if addr := ctx.EncodeContract(encodedAddr); addr != encodedAddr {
		t.Fatalf("Encoding contract failed")
	}
}

// TestContextDecodeContract encodes then decodes an address and
// compares whether the addresses are not the same.
func TestContextDecodeContract(t *testing.T) {
	ctx := NewReplay()
	encodedAddr := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if addr := ctx.EncodeContract(encodedAddr); addr != encodedAddr {
		t.Fatalf("Encoding contract failed")
	}
	decodedAddr := ctx.DecodeContract(encodedAddr)
	if encodedAddr != decodedAddr {
		t.Fatalf("Decoding contract failed")
	}
}

// TestContextPrevContract fetches the last used addresses
// after encoding and decoding, then compares whether they match the actual
// last used contract addresses.
func TestContextPrevContract(t *testing.T) {
	ctx := NewReplay()
	encodedAddr1 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1 := ctx.EncodeContract(encodedAddr1)
	lastAddr := ctx.PrevContract()
	if encodedAddr1 != lastAddr {
		t.Fatalf("Failed to get last contract address (1) after encoding")
	}

	encodedAddr2 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2 := ctx.EncodeContract(encodedAddr2)
	lastAddr = ctx.PrevContract()
	if encodedAddr2 != lastAddr {
		t.Fatalf("Failed to get last contract address (2) after encoding")
	}

	decodedAddr1 := ctx.DecodeContract(idx1)
	lastAddr = ctx.PrevContract()
	if decodedAddr1 != lastAddr {
		t.Fatalf("Failed to get last contract address (1) after decoding")
	}

	decodedAddr2 := ctx.DecodeContract(idx2)
	lastAddr = ctx.PrevContract()
	if decodedAddr2 != lastAddr {
		t.Fatalf("Failed to get last contract address (2) after decoding")
	}
}

// TestContextEncodeKey encodes a storage key and checks the returned index.
func TestContextEncodeKey(t *testing.T) {
	ctx := NewReplay()
	encodedKey := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	if _, idx := ctx.EncodeKey(encodedKey); idx != -1 {
		t.Fatalf("Encoding storage key failed; position: %d", idx)
	}
}

// TestContextDecodeKey encodes then decodes a storage key and compares
// whether the storage keys are not matched.
func TestContextDecodeKey(t *testing.T) {
	ctx := NewReplay()
	encodedKey := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	_, idx := ctx.EncodeKey(encodedKey)
	if idx != -1 {
		t.Fatalf("Encoding storage key failed; position: %d", idx)
	}
	decodedKey := ctx.DecodeKey(encodedKey)
	if encodedKey != decodedKey {
		t.Fatalf("Decoding storage key failed")
	}
}

// TestContextReadKeyCache reads storage key from index-cache after
// encoding/decoding new storage key. ReadKeyCache doesn't update top index.
func TestContextReadKeyCache(t *testing.T) {
	ctx := NewReplay()
	encodedKey1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1, _ := ctx.EncodeKey(encodedKey1)
	encodedKey2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2, _ := ctx.EncodeKey(encodedKey2)

	cachedKey := ctx.ReadKeyCache(1)
	if encodedKey1 != cachedKey {
		t.Fatalf("Failed to read storage key (1) from index-cache")
	}

	cachedKey = ctx.ReadKeyCache(0)
	if encodedKey2 != cachedKey {
		t.Fatalf("Failed to read storage key (2) from index-cache")
	}

	decodedKey1 := ctx.DecodeKey(idx1)
	decodedKey2 := ctx.DecodeKey(idx2)

	cachedKey = ctx.ReadKeyCache(1)
	if decodedKey1 != cachedKey {
		t.Fatalf("Failed to read storage key (1) from index-cache")
	}

	cachedKey = ctx.ReadKeyCache(0)
	if decodedKey2 != cachedKey {
		t.Fatalf("Failed to read storage key (2) from index-cache")
	}
}

// TestContextLookup reads storage key from index-cache after
// encoding/decoding new storage key. DecodeKeyCache updates top index.
func TestContextDecodeKeyCache(t *testing.T) {
	ctx := NewReplay()
	encodedKey1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	idx1, _ := ctx.EncodeKey(encodedKey1)
	encodedKey2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F274")
	idx2, _ := ctx.EncodeKey(encodedKey2)

	cachedKey := ctx.DecodeKeyCache(1)
	if encodedKey1 != cachedKey {
		t.Fatalf("Failed to lookup storage key (1) from index-cache")
	}

	cachedKey = ctx.DecodeKeyCache(1)
	if encodedKey2 != cachedKey {
		t.Fatalf("Failed to lookup storage key (2) from index-cache")
	}

	decodedKey1 := ctx.DecodeKey(idx1)
	decodedKey2 := ctx.DecodeKey(idx2)

	cachedKey = ctx.DecodeKeyCache(1)
	if decodedKey1 != cachedKey {
		t.Fatalf("Failed to lookup storage key (1) from index-cache")
	}

	cachedKey = ctx.DecodeKeyCache(1)
	if decodedKey2 != cachedKey {
		t.Fatalf("Failed to lookup storage key (2) from index-cache")
	}
}

// TestContextSnapshot adds a new snapshot pair to the snapshot
// context, then gets the replayed snapshot id from the context.
func TestContextSnapshot(t *testing.T) {
	ctx := NewReplay()
	recordedID := int32(39)
	replayedID1 := int32(50)
	ctx.AddSnapshot(recordedID, replayedID1)
	if ctx.GetSnapshot(recordedID) != replayedID1 {
		t.Fatalf("Failed to retrieve snapshot id")
	}
	replayedID2 := int32(8)
	ctx.AddSnapshot(recordedID, replayedID2)
	if ctx.GetSnapshot(recordedID) != replayedID2 {
		t.Fatalf("Failed to retrieve snapshot id")
	}
}
