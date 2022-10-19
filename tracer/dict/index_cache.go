package dict

import (
	"errors"
	"math"
)

// Length of cache (i.e. 2^8)
const CacheLength = 256

// IndexCache data structure for implementing a LRU cache
// policy.
type IndexCache struct {
	top  int                 // last accessed inde
	data [CacheLength]uint32 // data elements of cache
}

// ClearIndexCache clears the count queue by setting
// all data elements to MaxUint32, which is an an
// invalid index.
func (q *IndexCache) ClearIndexCache() {
	q.top = 0
	for i := 0; i < CacheLength; i++ {
		q.data[i] = math.MaxUint32
	}
}

// NewIndexCache creates a new queue for counting positions.
func NewIndexCache() *IndexCache {
	q := new(IndexCache)
	q.ClearIndexCache()
	return q
}

// Place puts a new element into cache.
func (q *IndexCache) Place(item uint32) int {
	// find the index in cache
	for i := 0; i < CacheLength; i++ {
		if q.data[i] == item {
			// calculate position in queue
			// relevant for encoding
			j := (q.top - i) % CacheLength
			if j < 0 {
				j += CacheLength
			}
			tmp := q.data[q.top]
			q.data[q.top] = q.data[i]
			q.data[i] = tmp
			return j
		}
	}
	// element is not found => place it as the most recent one
	q.top++
	q.top = q.top % CacheLength
	q.data[q.top] = item
	return -1
}

// Look up an element given a position in cache.
func (q *IndexCache) Lookup(pos int) (uint32, error) {
	if pos < 0 || pos >= CacheLength {
		return 0, errors.New("Position out of bound")
	}
	// calculate position in queue
	// relevant for encoding
	j := (q.top - pos) % CacheLength
	if j < 0 {
		j += CacheLength
	}
	if q.data[j] == math.MaxUint32 {
		return 0, errors.New("Undefined index in cache")
	}
	return q.data[j], nil
}
