package stochastic

import (
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/stat/distuv"
)

// containsQ checks whether an element is in the queue (ignoring the previous value).
func containsQ(slice []int, x int) bool {
	for i, n := range slice {
		if x == n && i != qstatsLen-1 {
			return true
		}
	}
	return false
}

// TestRandomAccessSimple tests random access generators for indexes.
func TestRandomAccessSimple(t *testing.T) {

	// create a random access index generator
	// with a zero probability distribution.
	qpdf := make([]float64, qstatsLen)
	ra := NewRandomAccess(1000, 5.0, qpdf)

	// check no argument class (must be always -1)
	if ra.NextIndex(noArgID) != -1 {
		t.Fatalf("expected an invalid index")
	}

	// check zero argument class (must be zero)
	if ra.NextIndex(zeroValueID) != 0 {
		t.Fatalf("expected an invalid index")
	}

	// check a new value (must be equal to the number of elements
	// in the index set and must be greater than zero).
	if idx := ra.NextIndex(newValueID); idx != ra.numElem || idx < 1 {
		t.Fatalf("expected a new index")
	}

	// check previous value (must return the first element in the queue
	// and the element + 1 is the returned value. The returned must be
	// in the range between 1 and ra.num).
	queue := make([]int, qstatsLen)
	copy(queue, ra.queue)
	if idx := ra.NextIndex(previousValueID); queue[0]+1 != idx || idx < 1 || idx > ra.numElem {
		t.Fatalf("accessing previous index failed")
	}

	// check recent value (must return an element in the queue excluding
	// the first element).
	copy(queue, ra.queue)
	if idx := ra.NextIndex(recentValueID); idx != -1 {
		t.Fatalf("index access must fail because no distribution was specified")
	}

	// create a uniform distribution for random generator and check recent access
	for i := 0; i < qstatsLen; i++ {
		qpdf[i] = 1.0 / float64(qstatsLen)
	}
	ra = NewRandomAccess(1000, 5.0, qpdf)
	copy(queue, ra.queue)
	if idx := ra.NextIndex(recentValueID); idx < 1 || idx > ra.numElem || !containsQ(queue, idx-1) {
		t.Fatalf("index access not in queue")
	}

	// check random access (must not be contained in queue)
	copy(queue, ra.queue)
	if idx := ra.NextIndex(randomValueID); idx < 1 || idx > ra.numElem || containsQ(queue, idx-1) || queue[0]+1 == idx {
		t.Fatalf("index access must fail because no distribution was specified")
	}
}

// TestQueuingStatsSimple tests previous accesses
func TestRandomAccessRecentAccess(t *testing.T) {

	// create a random access index generator
	// with a zero probability distribution.
	qpdf := make([]float64, qstatsLen)
	ra := NewRandomAccess(1000, 5.0, qpdf)

	// check a new value (must be equal to the number of elements
	// in the index set and must be greater than zero).
	idx1 := ra.NextIndex(newValueID)
	if idx1 != ra.numElem || idx1 < 1 {
		t.Fatalf("expected a new index")
	}
	idx2 := ra.NextIndex(previousValueID)
	if idx1 != idx2 {
		t.Fatalf("previous index access failed.")
	}
	idx3 := ra.NextIndex(previousValueID)
	if idx2 != idx3 {
		t.Fatalf("previous index access failed.")
	}
	// in the index set and must be greater than zero).
	idx4 := ra.NextIndex(newValueID)
	if idx4 != ra.numElem || idx4 < 1 {
		t.Fatalf("expected a new index")
	}
	idx5 := ra.NextIndex(previousValueID)
	if idx5 == idx3 {
		t.Fatalf("previous previous index access must not be identical.")
	}
}

// TestRandomAccessDeleteIndex tests deletion of an index
func TestRandomAcessDeleteIndex(t *testing.T) {
	// create a random access index generator
	// with a zero probability distribution.
	qpdf := make([]float64, qstatsLen)
	ra := NewRandomAccess(1000, 5.0, qpdf)
	idx := ra.NextIndex(previousValueID)
	if idx == -1 || idx < 1 || idx > ra.numElem {
		t.Fatalf("previous index access failed.")
	}

	// delete previous element
	ra.DeleteIndex(idx)
	if len(ra.queue) != qstatsLen {
		t.Fatalf("queue size did not stay constant.")
	}
	for _, x := range ra.queue {
		if x == idx {
			t.Fatalf("index stayed still in queue.")
		}
	}
	if ra.numElem != 999 {
		t.Fatalf("Cardinality of index set did not decrement.")
	}
}

// checkUniformQueueSelection performs a statistical test
// whether a queue with uniform position distribution is
// unbiased.
func checkUniformQueueSelection(qpdf []float64, numSteps int) bool {

	ra := NewRandomAccess(1000, 5.0, qpdf)

	// number of observed queue positions
	counts := make([]int, qstatsLen)

	// select numSteps queue position and count there occurrence
	for steps := 0; steps < numSteps; steps++ {
		idx := ra.getRandQPos()
		counts[idx]++
	}

	// first index must not be selected
	if counts[0] > 0 {
		return false
	}

	// compute chi-squared value for observations
	chi2 := float64(0.0)
	for i, v := range counts {
		if i != 0 {
			expected := float64(numSteps) * qpdf[i] / (1.0 - qpdf[0])
			err := expected - float64(v)
			// fmt.Printf("Err: %v %v\n", v, expected)
			chi2 += (err * err) / expected
		}
	}

	// Perform statistical test whether uniform queue distribution is unbiased
	// with an alpha of 0.05 and a degree of freedom of queue length minus two
	// (no first position!).
	alpha := 0.05
	df := float64(qstatsLen - 2)
	chi2Critical := distuv.ChiSquared{K: df, Src: nil}.Quantile(1.0 - alpha)
	// fmt.Printf("Chi^2 value: %v Chi^2 critical value: %v df: %v\n", chi2, chi2Critical, qstatsLen-2)

	return chi2 <= chi2Critical
}

// TestRandomAccessRandQPos checks the random selection of the queue position via a statistical test.
func TestRandomAccessRandQPos(t *testing.T) {
	// set random seed to make check determinsitic
	rand.Seed(4711)

	// create a uniform queue distribution
	qpdf := make([]float64, qstatsLen)
	for i := 0; i < qstatsLen; i++ {
		qpdf[i] = 1.0 / float64(qstatsLen)
	}

	// run statistical test
	if !checkUniformQueueSelection(qpdf, 100000) {
		t.Fatalf("The random queue selection for a uniform queue distribution is biased.")
	}

	// create a truncated geometric queue distribution
	alpha := 0.4
	for i := 0; i < qstatsLen; i++ {
		qpdf[i] = (1 - alpha) *
			math.Pow(alpha, qstatsLen) /
			(1.0 - math.Pow(alpha, qstatsLen)) *
			math.Pow(alpha, -float64(i+1))
	}

	// run statistical test
	if !checkUniformQueueSelection(qpdf, 100000) {
		t.Fatalf("The random queue selection for truncated geometric queue distribution is biased.")
	}
}

// TestRandomAccessRandInd checks the random selection of the queue position via a statistical test.
func TestRandomAccessRandInd(t *testing.T) {
	// parameters
	lambda := 5.0
	numSteps := 10000
	idxRange := 10

	// populate buckets
	counts := make([]int, idxRange)
	for steps := 0; steps < numSteps; steps++ {
		counts[randIndex(lambda, idxRange)]++
	}

	// compute chi-squared value for observations
	chi2 := float64(0.0)
	for i, v := range counts {
		// compute expected value of bucket
		p := Cdf(lambda, float64(i+1)/float64(idxRange)) - Cdf(lambda, float64(i)/float64(idxRange))
		expected := float64(numSteps) * p
		err := expected - float64(v)
		chi2 += (err * err) / expected
		// fmt.Printf("Err: %v %v\n", v, expected)
	}

	// Perform statistical test whether uniform queue distribution is unbiased
	// with an alpha of 0.05 and a degree of freedom of queue length minus two
	// (no first position!).
	alpha := 0.05
	df := float64(idxRange - 1)
	chi2Critical := distuv.ChiSquared{K: df, Src: nil}.Quantile(1.0 - alpha)
	// fmt.Printf("Chi^2 value: %v Chi^2 critical value: %v df: %v\n", chi2, chi2Critical, df)

	if chi2 > chi2Critical {
		t.Fatalf("The random index selection biased.")
	}
}
