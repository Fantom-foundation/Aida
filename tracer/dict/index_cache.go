package dict

import (
	"fmt"
	"math"
)

// IndexCacheLength sets the length of index cache (i.e. 2^8)
const IndexCacheLength = 256

// IndexCache data structure for implementing an LRU cache policy.
type IndexCache struct {
	top  int                      // last accessed index
	data [IndexCacheLength]uint32 // indexes of cache
}

// Clear the index cache by setting all indexes to MaxUint32
// (representing an invalid index).
func (q *IndexCache) Clear() {
	q.top = 0
	for i := 0; i < IndexCacheLength; i++ {
		q.data[i] = math.MaxUint32
	}
}

// NewIndexCache creates a new index cache.
func NewIndexCache() *IndexCache {
	q := new(IndexCache)
	q.Clear()
	return q
}

// Place puts a new index into the index cache.
func (q *IndexCache) Place(item uint32) int {
	// find the index in cache
	for i := 0; i < IndexCacheLength; i++ {
		if q.data[i] == item {
			// calculate position in queue
			// relevant for encoding
			j := (q.top - i) % IndexCacheLength
			if j < 0 {
				j += IndexCacheLength
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
	// index is not found => place it as the most recent one
	q.top++
	q.top = q.top % IndexCacheLength
	q.data[q.top] = item
	// return an undefined position
	return -1
}

// Get an index for a cache position.
func (q *IndexCache) Get(pos int) (uint32, error) {
	if pos < 0 || pos >= IndexCacheLength {
		return 0, fmt.Errorf("Position %v out of bound", pos)
	}
	// calculate position in index cache
	j := (q.top - pos) % IndexCacheLength
	if j < 0 {
		j += IndexCacheLength
	}
	// check index
	if q.data[j] == math.MaxUint32 {
		return 0, fmt.Errorf("Position %v is an undefined index", pos)
	}
	// return index
	return q.data[j], nil
}
