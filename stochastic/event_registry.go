package stochastic

import (
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/simplify"

	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// EventRegistry counts events and counts transition for the Markov-Process.
type EventRegistry struct {
	// Frequency of argument-encoded operations
	argOpFreq [numArgOps]uint64

	// Transition frequencies between two subsequent argument-encoded operations
	transitFreq [numArgOps][numArgOps]uint64

	// Contract-address access statistics
	contracts statistics.Access[common.Address]

	// Storage-key access statistics
	keys statistics.Access[common.Hash]

	// Storage-value access statistics
	values statistics.Access[common.Hash]

	// Previous argument-encoded operation
	prevArgOp int

	// Snapshot deltas
	snapshotFreq map[int]uint64
}

// EventRegistryJSON is the JSON output for an event registry.
type EventRegistryJSON struct {
	Operations       []string    `json:"operations"`       // name of operations with argument classes
	StochasticMatrix [][]float64 `json:"stochasticMatrix"` // observed stochastic matrix

	// access statistics for contracts, keys, and values
	Contracts statistics.AccessJSON `json:"contractStats"`
	Keys      statistics.AccessJSON `json:"keyStats"`
	Values    statistics.AccessJSON `json:"valueSats"`

	// snapshot delta frequencies
	SnapshotEcdf [][2]float64 `json:"snapshotEcdf"`
}

// NewEventRegistry creates a new event registry.
func NewEventRegistry() EventRegistry {
	return EventRegistry{
		prevArgOp:    numArgOps,
		contracts:    statistics.NewAccess[common.Address](),
		keys:         statistics.NewAccess[common.Hash](),
		values:       statistics.NewAccess[common.Hash](),
		snapshotFreq: map[int]uint64{},
	}
}

// RegisterOp registers an operation with no simulation arguments
func (r *EventRegistry) RegisterOp(op int) {
	if op < 0 || op >= NumOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass := statistics.NoArgID
	keyClass := statistics.NoArgID
	valueClass := statistics.NoArgID

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFreq(op, addrClass, keyClass, valueClass)
}

// RegisterAddressOp registers an operation with a contract-address argument
func (r *EventRegistry) RegisterAddressOp(op int, address *common.Address) {
	// check ID
	if op < 0 || op >= NumOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass := r.contracts.Classify(*address)
	keyClass := statistics.NoArgID
	valueClass := statistics.NoArgID

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFreq(op, addrClass, keyClass, valueClass)

	// update contract address for subsequent classification
	r.contracts.Place(*address)
}

// RegisterAddressKeyOp registers an operation with a contract-address and a storage-key arguments.
func (r *EventRegistry) RegisterKeyOp(op int, address *common.Address, key *common.Hash) {
	// check operation range
	if op < 0 || op >= NumOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass := r.contracts.Classify(*address)
	keyClass := r.keys.Classify(*key)
	valueClass := statistics.NoArgID

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFreq(op, addrClass, keyClass, valueClass)

	// update contract address and storage key for subsequent classification
	r.contracts.Place(*address)
	r.keys.Place(*key)
}

// RegisterAddressKeyOp registers an operation with a contract-address, a storage-key and storage-value arguments.
func (r *EventRegistry) RegisterValueOp(op int, address *common.Address, key *common.Hash, value *common.Hash) {
	// check operation range
	if op < 0 || op >= NumOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass := r.contracts.Classify(*address)
	keyClass := r.keys.Classify(*key)
	valueClass := r.values.Classify(*value)

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFreq(op, addrClass, keyClass, valueClass)

	// update contract address and storage key for subsequent classification
	r.contracts.Place(*address)
	r.keys.Place(*key)
	r.values.Place(*value)
}

// updateFreq updates operation and transition frequency.
func (r *EventRegistry) updateFreq(op int, addr int, key int, value int) {
	// encode argument classes to compute specialized operation using a Horner's scheme
	argOp := EncodeArgOp(op, addr, key, value)

	// increment operation's frequency depending on argument class
	r.argOpFreq[argOp]++

	// skip counting for the first operation (there is no previous one)
	if r.prevArgOp < numArgOps {
		r.transitFreq[r.prevArgOp][argOp] = r.transitFreq[r.prevArgOp][argOp] + 1
	}
	r.prevArgOp = argOp
}

// RegisterSnapshotDelta counts the delta of a snapshot. The delta is the
// the number of stack elements between the top-of-stack of the snapshot stack
// and the reverted snapshot. A delta of zero means that the state is reverted
// to the previously created snapshot, a delta of one means that the state is
// reverted to the previous of the previous snapshot and so on.
func (r *EventRegistry) RegisterSnapshotDelta(delta int) {
	r.snapshotFreq[delta]++
}

// NewEventRegistry produces the JSON output for an event registry.
func (r *EventRegistry) NewEventRegistryJSON() EventRegistryJSON {
	// generate labels for observable operations
	label := []string{}
	for argop := 0; argop < numArgOps; argop++ {
		if r.argOpFreq[argop] > 0 {
			// decode argument-encoded operation
			op, addr, key, value := DecodeArgOp(argop)
			label = append(label, EncodeOpcode(op, addr, key, value))
		}
	}

	// Compute stochastic matrix for observable operations with their arguments
	A := [][]float64{}
	for i := 0; i < numArgOps; i++ {
		if r.argOpFreq[i] > 0 {
			row := []float64{}
			// find row total of row (i.e. state i)
			total := uint64(0)
			for j := 0; j < numArgOps; j++ {
				total += r.transitFreq[i][j]
			}
			// normalize row
			for j := 0; j < numArgOps; j++ {
				if r.argOpFreq[j] > 0 {
					row = append(row, float64(r.transitFreq[i][j])/float64(total))
				}
			}
			A = append(A, row)
		}
	}

	// Compute ECDF of snapshot deltas
	totalFreq := uint64(0)
	maxDelta := 0
	for delta, freq := range r.snapshotFreq {
		totalFreq += freq
		if maxDelta < delta {
			maxDelta = delta
		}
	}
	// simplified eCDF
	var simplified orb.LineString

	// if no data-points, nothing to plot
	if len(r.snapshotFreq) > 0 {

		// construct full eCdf as LineString
		ls := orb.LineString{}

		// print points of the empirical cumulative freq
		sumP := float64(0.0)

		// Correction term for Kahan's sum
		cP := float64(0.0)

		// add first point to line string
		ls = append(ls, orb.Point{0.0, 0.0})

		// iterate through all deltas
		for delta := 0; delta <= maxDelta; delta++ {
			// Implement Kahan's summation to avoid errors
			// for accumulated probabilities (they might be very small)
			// https://en.wikipedia.org/wiki/Kahan_summation_algorithm
			f := float64(r.snapshotFreq[delta]) / float64(totalFreq)
			x := float64(delta) / float64(maxDelta)

			yP := f - cP
			tP := sumP + yP
			cP = (tP - sumP) - yP
			sumP = tP

			// add new point to Ecdf
			ls = append(ls, orb.Point{x, sumP})
		}

		// add last point
		ls = append(ls, orb.Point{1.0, 1.0})

		// reduce full ecdf using Visvalingam-Whyatt algorithm to
		// "numPoints" points. See:
		// https://en.wikipedia.org/wiki/Visvalingam-Whyatt_algorithm
		simplifier := simplify.VisvalingamKeep(statistics.NumDistributionPoints)
		simplified = simplifier.Simplify(ls).(orb.LineString)
	}
	// convert orb.LineString to [][2]float64
	eCdf := make([][2]float64, len(simplified))
	for i := range simplified {
		eCdf[i] = [2]float64(simplified[i])
	}

	return EventRegistryJSON{
		Operations:       label,
		StochasticMatrix: A,
		Contracts:        r.contracts.NewAccessJSON(),
		Keys:             r.keys.NewAccessJSON(),
		Values:           r.values.NewAccessJSON(),
		SnapshotEcdf:     eCdf,
	}
}
