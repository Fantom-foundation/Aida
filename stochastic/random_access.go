package stochastic

import (
	"fmt"
	"math"
	"math/rand"
)

// minRandomAccessSize must be substantially larger than qstatsLen
// (Otherwise sampling for arguments with class RandomValueID may
// take a very long time and would slow down the simulation.)
const minRandomAccessSize = 10 * qstatsLen

// RandomAccess data structure for producing random index accesses.
type RandomAccess struct {
	// cardinality of set
	numElem int64

	// lambda parameter of exponential distribution
	lambda float64

	// queue for indexes (always fixed to length qStatslen)
	// Note that elements in queue are stored in the range from 0 to numElem-1
	// and need to be shifted by one due to the zero value.
	queue []int64

	// probability distribution of queue for selecting recent values.
	qpdf []float64
}

// NewAccessStats creates a new access index.
func NewRandomAccess(numElem int64, lambda float64, qpdf []float64) *RandomAccess {
	if numElem < minRandomAccessSize {
		return nil
	}

	// fill queue with uniform random indexes.
	queue := []int64{}
	for i := 0; i < qstatsLen; i++ {
		queue = append(queue, rand.Int63n(numElem))
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
func (a *RandomAccess) NextIndex(class int) int64 {
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
		if a.numElem == math.MaxInt64 {
			return -1
		}
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
		// return the value of the first position in the queue
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
func (a *RandomAccess) DeleteIndex(v int64) error {
	// check index;
	if v <= 0 || v > a.numElem {
		// NB: cannot delete zero index!
		return fmt.Errorf("DeleteIndex: wrong index range")
	}

	// reduce cardinality by one
	a.numElem--
	if a.numElem < minRandomAccessSize {
		return fmt.Errorf("DeleteIndex: cardinality of set too low")
	}

	// replace deleted index by a random index
	// (necessary only in case if the deleted element
	// is the last element of the set)
	j := randIndex(a.lambda, a.numElem)
	for i := 0; i < qstatsLen; i++ {
		if a.queue[i]+1 == v {
			a.queue[i] = j
		}
	}

	return nil
}

// findQElem finds an element in the queue.
func (a *RandomAccess) findQElem(elem int64) bool {
	for i := 0; i < qstatsLen; i++ {
		if a.queue[i] == elem {
			return true
		}
	}
	return false
}

// randIndex produces an index between 0 and n-1 using
// an exponential distribution with parameter lambda n.
func randIndex(lambda float64, n int64) int64 {
	return int64(float64(n) * invCdf(lambda, rand.Float64()))
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
	// use Kahan's sum for avoiding numerical issues.
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
func (a *RandomAccess) placeQ(elem int64) {
	a.queue = append([]int64{elem}, a.queue[0:qstatsLen-1]...)
}

// lastQ returns previously queued element.
func (a *RandomAccess) lastQ() int64 {
	return a.queue[0]
}

// recentQ returns some element in the queue but not the previous one.
func (a *RandomAccess) recentQ() int64 {
	i := a.getRandQPos()
	switch i {
	case 0:
		panic("getRandPos() returned previous element.")
	case -1:
		return -1
	default:
		return a.queue[i]
	}
}
