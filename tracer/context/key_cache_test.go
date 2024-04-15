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
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// makeRandomByteSlice creates byte slice of given length with randomized values
func makeRandomByteSlice(t *testing.T, bufferLength int) []byte {
	// make byte slice
	buffer := make([]byte, bufferLength)

	// fill the slice with random data
	_, err := rand.Read(buffer)
	if err != nil {
		t.Fatalf("failed test data; can not generate random byte slice; %s", err.Error())
	}

	return buffer
}

// TestKeyCacheSimple tests for existence of keys in the key cache and
// checks the positions of get before and after clearing the key cache.
func TestKeyCacheSimple(t *testing.T) {
	zeroHash := common.Hash{}
	testedValue1 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	testedValue2 := common.HexToHash("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")

	// create key cache
	cache := NewKeyCache()

	// place first key
	pos := cache.Place(testedValue1)

	// position should be -1 (does not exist)
	if pos != -1 {
		t.Fatalf("First key must not exist and must return position -1(=undef).")
	}

	// place first key again
	pos = cache.Place(testedValue1)
	if pos != 0 {
		t.Fatalf("First key must exist and must return position 0")
	}

	// place second key
	pos = cache.Place(testedValue2)
	if pos != -1 {
		t.Fatalf("Second key must not exist and must return position -1(=undef)")
	}

	// place second key again
	pos = cache.Place(testedValue2)
	if pos != 0 {
		t.Fatalf("Second key must exist and must return position 0")
	}

	// get the position of first and second key
	pos1hash, err1 := cache.Get(0)
	if err1 != nil {
		t.Fatalf("Get of first key failed. Error: %v", err1)
	}
	pos2hash, err2 := cache.Get(1)
	if err2 != nil {
		t.Fatalf("Get of second key failed. Error: %v", err2)
	}
	if pos1hash != testedValue2 && pos2hash != testedValue1 {
		t.Fatalf("Get has changed key cache.")
	}

	// clear cache
	cache.Clear()

	// execute get again and check that the invocations of get are failing
	hash1, err1 := cache.Get(0)
	if err1 != nil && hash1 == zeroHash {
		t.Fatalf("Get of first key must return zero hash: %v", err1)
	}
	hash2, err2 := cache.Get(1)
	if err2 != nil && hash2 == zeroHash {
		t.Fatalf("Get of second key must return zero hash: %v", err2)
	}
	_, err3 := cache.Get(-1)
	if err3 == nil {
		t.Fatalf("Get of key out of range must fail.")
	}
}

// TestCachekeyOverflow tests that the least recently used item is evicted.
func TestKeyCacheOverflow(t *testing.T) {
	// create key cache
	cache := NewKeyCache()

	// create random hash and place it in cache
	firstCacheItem := common.BytesToHash(makeRandomByteSlice(t, 40))
	cache.Place(firstCacheItem)

	// place 256 - 1 keys
	for i := 0; i <= KeyCacheLength-1; i++ {
		randomHash := common.BytesToHash(makeRandomByteSlice(t, 40))
		cache.Place(randomHash)
	}

	// place first key again, and check for eviction
	pos := cache.Place(firstCacheItem)
	if pos != -1 {
		t.Fatalf("First key must have been evicted.")
	}
}
