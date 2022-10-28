package dict

import (
	"errors"
	"math"
)

// CacheLength sets the length of the cache (i.e. 2^8).
const CacheLength = 256

// IndexCache data structure
type IndexCache struct {
	top  int                 // last accessed element
	data [CacheLength]uint32 // data elements of index cache
}

// ClearIndexCache clears the index cache.
func (q *IndexCache) ClearIndexCache() {
	q.top = 0
	for i := 0; i < CacheLength; i++ {
		q.data[i] = math.MaxUint32
	}
}

// NewIndexCache creates an index cache.
func NewIndexCache() *IndexCache {
	q := new(IndexCache)
	q.ClearIndexCache()
	return q
}

// Place puts a new element into index cache.
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

// Lookup an element in the index cache.
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
