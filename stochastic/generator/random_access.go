package generator

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/Fantom-foundation/Aida/stochastic/exponential"
	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// MinRandomAccessSize must be substantially larger than statistics.QueueLen
// (Otherwise sampling for arguments with class RandomValueID may
// take a very long time and would slow down the simulation.)
const MinRandomAccessSize = 10 * statistics.QueueLen

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

	// random generator
	rg *rand.Rand
}

// NewAccess creates a new access index.
func NewRandomAccess(rg *rand.Rand, numElem int64, lambda float64, qpdf []float64) *RandomAccess {
	if numElem < MinRandomAccessSize {
		return nil
	}

	// fill queue with uniform random indexes.
	queue := []int64{}
	for i := 0; i < statistics.QueueLen; i++ {
		queue = append(queue, rg.Int63n(numElem))
	}

	// create a copy of the queue distribution.
	copyQpdf := make([]float64, statistics.QueueLen)
	copy(copyQpdf, qpdf)

	return &RandomAccess{
		numElem: numElem,
		lambda:  lambda,
		queue:   queue,
		qpdf:    copyQpdf,
		rg:      rg,
	}
}

// NextIndex returns the next random index based on the provided class.
func (a *RandomAccess) NextIndex(class int) int64 {
	switch class {

	case statistics.NoArgID:
		return -1

	case statistics.ZeroValueID:
		// only way to return zero value/all other access classes
		// will result in a non-zero result.
		return 0

	case statistics.NewValueID:
		// increment population size of access set
		// and return newly introduced element.
		if a.numElem == math.MaxInt64 {
			return -1
		}
		v := a.numElem
		a.placeQ(v)
		a.numElem++
		return v + 1

	case statistics.RandomValueID:
		// use randomised value that is not contained in the queue
		for {
			v := exponential.DiscreteSample(a.rg, a.lambda, a.numElem)
			if !a.findQElem(v) {
				a.placeQ(v)
				return v + 1
			}
		}

	case statistics.PreviousValueID:
		// return the value of the first position in the queue
		v := a.lastQ()
		a.placeQ(v)
		return v + 1

	case statistics.RecentValueID:
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
	// check index range
	if v < 0 || v >= a.numElem {
		return fmt.Errorf("DeleteIndex: index (%v) out of index range", v)
	}

	// reduce cardinality by one
	a.numElem--
	if a.numElem < MinRandomAccessSize {
		return fmt.Errorf("DeleteIndex: cardinality of set too low")
	}

	// replace deleted last element by new element
	// note that the actual deleted element may be
	// in range, but there might elements in the queue
	// that exceed the new range limit. They need to
	// be replaced.
	j := exponential.DiscreteSample(a.rg, a.lambda, a.numElem)
	for i := 0; i < statistics.QueueLen; i++ {
		if a.queue[i] >= a.numElem {
			a.queue[i] = j
		}
	}

	return nil
}

// findQElem finds an element in the queue.
func (a *RandomAccess) findQElem(elem int64) bool {
	for i := 0; i < statistics.QueueLen; i++ {
		if a.queue[i] == elem {
			return true
		}
	}
	return false
}

// getRandQPos obtains the next queue position.
// TODO: Consider replacing the pdf with an
// exponential distribution. A new value can be
// produced much faster.
func (a *RandomAccess) getRandQPos() int {
	// obtain random number in [0, 1.0)
	r := a.rg.Float64()

	// compute inverse CDF and select the index
	sum := float64(0)
	c := float64(0)
	factor := 1.0 - a.qpdf[0]
	j := -1
	// skip first slot (only used for previousValue)
	// use Kahan's sum for avoiding numerical issues.
	for i := 1; i < statistics.QueueLen; i++ {
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
	a.queue = append([]int64{elem}, a.queue[0:statistics.QueueLen-1]...)
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
