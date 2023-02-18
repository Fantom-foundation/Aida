package stochastic

import (
	"fmt"
	"log"
	"math"
	"math/rand"
)

// RandomAccess data structure for producing random accesses.
type RandomAccess struct {
	// cardinality of set
	numElem int

	// lambda parameter of exponential distribution
	lambda float64

	// queue for indexes  (always fixed length qStatslen)
	queue []int

	// probability distribution of queue (always fixed length qStatslen)
	qpdf []float64
}

// NewAccessStats creates a new access index.
func NewRandomAccess(numElem int, lambda float64, qpdf []float64) *RandomAccess {
	if numElem < qstatsLen {
		log.Fatalf("NewRandomAccess: number of elements smaller than the queue length.")
	}

	// fill queue with uniform random indexes.
	queue := []int{}
	for i := 0; i < qstatsLen; i++ {
		queue = append(queue, rand.Intn(numElem))
	}

	// create a copy of the queue distribution.
	copyQpdf := make([]float64, qstatsLen)
	copy(copyQpdf, qpdf)

	return &RandomAccess{
		numElem: numElem,
		lambda:  lambda,
		queue:   queue,
		qpdf:    copyQpdf,
	}
}

// NextIndex returns the next random index based on the provided class.
func (a *RandomAccess) NextIndex(class int) int {
	switch class {

	case noArgID:
		return -1

	case zeroValueID:
		// only way to return zero value/all other access classes
		// will result in a non-zero result.
		return 0

	case newValueID:
		// increment population size of access set
		// and return newly introduced element.
		v := a.numElem
		a.placeQ(v)
		a.numElem++
		return v + 1

	case randomValueID:
		// use randomised value that is not contained in the queue
		for {
			v := randIndex(a.lambda, a.numElem)
			if !a.findQElem(v) {
				a.placeQ(v)
				return v + 1
			}
		}

	case previousValueID:
		// return the value in the first position in the queue
		v := a.lastQ()
		a.placeQ(v)
		return v + 1

	case recentValueID:
		if v := a.recentQ(); v != -1 {
			a.placeQ(v)
			return v + 1
		} else {
			return -1
		}

	default:
		return -1
	}
}

// DeleteIndex deletes an access index.
func (a *RandomAccess) DeleteIndex(v int) error {
	// check index
	if v <= 0 || v >= a.numElem {
		return fmt.Errorf("DeleteIndex: wrong index range")
	}

	// reduce cardinality of set by one
	a.numElem--
	if a.numElem < qstatsLen {
		return fmt.Errorf("DeleteIndex: cardinality of set too low")
	}

	// replaced deleted index by a random index
	j := randIndex(a.lambda, a.numElem)
	for i := 0; i < qstatsLen; i++ {
		if a.queue[i] == v {
			a.queue[i] = j
		}
	}

	return nil
}

// findQElem finds an element in the queue.
func (a *RandomAccess) findQElem(elem int) bool {
	for i := 0; i < qstatsLen; i++ {
		if a.queue[i] == elem {
			return true
		}
	}
	return false
}

// randIndex produces an index between 0 and n-1 using
// an exponential distribution with parameter lambda n.
func randIndex(lambda float64, n int) int {
	return int(float64(n) * invCdf(lambda, rand.Float64()))
}

// invCdf is the inverse cumulative distribution function for
// producing random values following the exponential distribution
// with parameter lambda (providing probability p).
func invCdf(lambda float64, p float64) float64 {
	return math.Log(p*math.Exp(-lambda)-p+1) / -lambda
}

// getRandQPos obtains the next queue position.
func (a *RandomAccess) getRandQPos() int {
	// obtain random number in [0, 1.0)
	r := rand.Float64()

	// compute inverse CDF and select the index
	sum := float64(0)
	c := float64(0)
	factor := 1.0 - a.qpdf[0]
	j := -1
	// skip first slot (only used for previousValue)
	for i := 1; i < qstatsLen; i++ {
		y := (a.qpdf[i] / factor) - c
		t := sum + y
		c = (t - sum) - y
		sum = t
		if r <= sum {
			return i
		}
		// if numerically unstable, return last
		// non-zero entry as a solution.
		if a.qpdf[i] > 0.0 {
			j = i
		}
	}
	return j
}

// placeQ places element in the queue.
func (a *RandomAccess) placeQ(elem int) {
	a.queue = append([]int{elem}, a.queue[0:qstatsLen-1]...)
}

// lastQ return previously queued element.
func (a *RandomAccess) lastQ() int {
	return a.queue[0]
}

// recentQ return some element in the queue but not the previous one.
func (a *RandomAccess) recentQ() int {
	if idx := a.getRandQPos(); idx != -1 {
		return a.queue[a.getRandQPos()]
	} else {
		return -1
	}
}
