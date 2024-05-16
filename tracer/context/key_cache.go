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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// KeyCacheLength sets the length of key cache (i.e. 2^8)
const KeyCacheLength = 256

// KeyCache data structure for keeping a cache of most frequent accessed storage keys
type KeyCache struct {
	top  int                         // last accessed key
	data [KeyCacheLength]common.Hash // keys in cache
}

// Clear the key cache by setting all keys to null.
func (q *KeyCache) Clear() {
	q.top = 0
	for i := 0; i < KeyCacheLength; i++ {
		q.data[i] = common.Hash{}
	}
}

// NewKeyCache creates a new key cache.
func NewKeyCache() *KeyCache {
	return &KeyCache{}
}

// Place puts a new key into the key cache.
func (q *KeyCache) Place(item common.Hash) int {
	// find a key in the cache
	for i := 0; i < KeyCacheLength; i++ {
		if q.data[i] == item {
			// calculate position in queue
			// relevant for encoding
			j := (q.top - i) % KeyCacheLength
			if j < 0 {
				j += KeyCacheLength
			}
			// Note that we don't preserve the temporal
			// order in the cache by the following triangular swap.
			// However, experiments showed that the gains
			// preserving the temporal order inside the cahce
			// are small (<= 0.01%).
			tmp := q.data[q.top]
			q.data[q.top] = q.data[i]
			q.data[i] = tmp
			return j
		}
	}
	// key is not found => place it as the most recent one
	q.top++
	q.top = q.top % KeyCacheLength
	q.data[q.top] = item
	// return an undefined position
	return -1
}

// Get a key for a cache position.
func (q *KeyCache) Get(pos int) (common.Hash, error) {
	if pos < 0 || pos >= KeyCacheLength {
		return common.Hash{}, fmt.Errorf("Position %v out of bound", pos)
	}
	// calculate position in key cache
	j := (q.top - pos) % KeyCacheLength
	if j < 0 {
		j += KeyCacheLength
	}
	// return key
	return q.data[j], nil
}
