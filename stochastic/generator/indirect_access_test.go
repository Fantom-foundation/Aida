package generator

import (
	"testing"

	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// containsQ checks whether an element is in the queue (ignoring the previous value).
func containsIndirectQ(slice []int64, x int64) bool {
	for i, n := range slice {
		if x == n && i != 0 {
			return true
		}
	}
	return false
}

// TestIndirectAccessSimple tests indirect access generator for indexes.
func TestIndirectAccessSimple(t *testing.T) {

	// create a random access index generator
	// with a zero probability distribution.
	qpdf := make([]float64, statistics.QueueLen)
	ia := NewIndirectAccess(NewRandomAccess(1000, 5.0, qpdf))

	// check no argument class (must be always -1)
	if ia.NextIndex(statistics.NoArgID) != -1 {
		t.Fatalf("expected an invalid index")
	}

	// check zero argument class (must be zero)
	if ia.NextIndex(statistics.ZeroValueID) != 0 {
		t.Fatalf("expected an invalid index")
	}

	// check a new value (must be equal to the number of elements
	// in the index set and must be greater than zero).
	if idx := ia.NextIndex(statistics.NewValueID); idx != ia.NumElem() || idx < 1 {
		t.Fatalf("expected a new index (%v, %v)", idx, ia.NumElem())
	}

	// run check again.
	if idx := ia.NextIndex(statistics.NewValueID); idx != ia.NumElem() || idx < 1 {
		t.Fatalf("expected a new index (%v, %v)", idx, ia.NumElem())
	}

	// check previous value (must return the first element in the queue
	// and the element + 1 is the returned value. The returned must be
	// in the range between 1 and ra.num).
	queue := make([]int64, statistics.QueueLen)
	copy(queue, ia.randAcc.queue)
	if idx := ia.NextIndex(statistics.PreviousValueID); ia.translation[ia.randAcc.lastQ()] != idx || idx < 1 || idx > ia.NumElem() {
		t.Fatalf("accessing previous index failed (%v, %v)", idx, ia.translation[ia.randAcc.lastQ()])
	}

	// check recent value (must return an element in the queue excluding
	// the first element).
	copy(queue, ia.randAcc.queue)
	if idx := ia.NextIndex(statistics.RecentValueID); idx != -1 {
		t.Fatalf("index access must fail because no distribution was specified")
	}

	// create a uniform distribution for random generator and check recent access
	for i := 0; i < statistics.QueueLen; i++ {
		qpdf[i] = 1.0 / float64(statistics.QueueLen)
	}

	ia = NewIndirectAccess(NewRandomAccess(1000, 5.0, qpdf))
	copy(queue, ia.randAcc.queue)
	if idx := ia.NextIndex(statistics.RecentValueID); idx < 1 || idx > ia.NumElem() || !containsIndirectQ(queue, idx-1) {
		t.Fatalf("index access not in queue")
	}

	// check random access (must not be contained in queue)
	copy(queue, ia.randAcc.queue)
	if idx := ia.NextIndex(statistics.RandomValueID); idx < 1 || idx > ia.NumElem() || containsIndirectQ(queue, idx-1) || queue[0]+1 == idx {
		t.Fatalf("index access must fail because no distribution was specified")
	}
}

// TestIndirectAccessRecentAccess tests previous accesses
func TestIndirectAccessRecentAccess(t *testing.T) {

	// create a random access index generator
	// with a zero probability distribution.
	qpdf := make([]float64, statistics.QueueLen)
	ra := NewRandomAccess(1000, 5.0, qpdf)
	ia := NewIndirectAccess(ra)

	// check a new value (must be equal to the number of elements
	// in the index set and must be greater than zero).
	idx1 := ia.NextIndex(statistics.NewValueID)
	if idx1 != ra.numElem || idx1 < 1 {
		t.Fatalf("expected a new index")
	}
	idx2 := ia.NextIndex(statistics.PreviousValueID)
	if idx1 != idx2 {
		t.Fatalf("previous index access failed. (%v, %v)", idx1, idx2)
	}
	idx3 := ia.NextIndex(statistics.PreviousValueID)
	if idx2 != idx3 {
		t.Fatalf("previous index access failed.")
	}
	// in the index set and must be greater than zero).
	idx4 := ia.NextIndex(statistics.NewValueID)
	if idx4 != ra.numElem || idx4 < 1 {
		t.Fatalf("expected a new index")
	}
	idx5 := ia.NextIndex(statistics.PreviousValueID)
	if idx5 == idx3 {
		t.Fatalf("previous previous index access must not be identical.")
	}
}

// TestIndirectAccessDeleteIndex tests deletion of an index
func TestIndirectAcessDeleteIndex(t *testing.T) {
	// create a random access index generator
	// with a zero probability distribution.
	qpdf := make([]float64, statistics.QueueLen)
	ra := NewRandomAccess(1000, 5.0, qpdf)
	ia := NewIndirectAccess(ra)
	idx := int64(500) // choose an index in the middle of the range

	// delete previous element
	err := ia.DeleteIndex(idx)
	if err != nil {
		t.Fatalf("Deletion failed.")
	}

	// check whether index still exists
	for i := int64(0); i < ia.NumElem(); i++ {
		if ia.translation[i] == idx {
			t.Fatalf("index still exists.")
		}
	}
}
