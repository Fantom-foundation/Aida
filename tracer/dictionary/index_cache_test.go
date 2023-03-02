package dictionary

import (
	"testing"
)

// TestIndexCacheSimple tests for existence of indexes in the index cache and
// checks the positions of get before and after clearing the index cache.
func TestIndexCacheSimple(t *testing.T) {
	// create index cache
	cache := NewIndexCache()

	// place first index
	pos := cache.Place(0)

	// position should be -1 (does not exist)
	if pos != -1 {
		t.Fatalf("First index must not exist and must return position -1(=undef).")
	}

	// place first index again
	pos = cache.Place(0)
	if pos != 0 {
		t.Fatalf("First index must exist and must return position 0")
	}

	// place second index
	pos = cache.Place(1)
	if pos != -1 {
		t.Fatalf("Second index must not exist and must return position -1(=undef)")
	}

	// place second index again
	pos = cache.Place(1)
	if pos != 0 {
		t.Fatalf("Second index must exist and must return position 0")
	}

	// get the position of first and second index
	pos1, err1 := cache.Get(0)
	if err1 != nil {
		t.Fatalf("Get of first index failed. Error: %v", err1)
	}
	pos2, err2 := cache.Get(1)
	if err2 != nil {
		t.Fatalf("Get of second index failed. Error: %v", err2)
	}
	if pos1 != 1 && pos2 != 0 {
		t.Fatalf("Get has changed index cache.")
	}

	// clear cache
	cache.Clear()

	// execute get again and check that the invocations of get are failing
	_, err1 = cache.Get(0)
	if err1 == nil {
		t.Fatalf("Get of first index must fail.")
	}
	_, err2 = cache.Get(1)
	if err2 == nil {
		t.Fatalf("Get of second index must fail.")
	}
}

// TestCacheIndexOverflow tests that the least recently used item is evicted.
func TestIndexCacheOverflow(t *testing.T) {
	// create index cache
	cache := NewIndexCache()

	// place 256 indexes
	for i := uint32(0); i <= IndexCacheLength; i++ {
		cache.Place(i)
	}

	// place first index again, and check for eviction
	pos := cache.Place(0)
	if pos != -1 {
		t.Fatalf("First index must have been evicted.")
	}
}
