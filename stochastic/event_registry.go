package stochastic

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
)

// numArgOps gives the number of operations with encoded argument classes
const numArgOps = numOps * numClasses * numClasses * numClasses

// EventRegistry counts events and counts transition for the Markov-Process.
type EventRegistry struct {
	// Frequency of argument-encoded operations
	argOpFreq [numArgOps]uint64

	// Transition frequencies between two subsequent argument-encoded operations
	transitFreq [numArgOps][numArgOps]uint64

	// Contract-address access statistics
	contracts AccessStats[common.Address]

	// Storage-key access statistics
	keys AccessStats[common.Hash]

	// Storage-value access statistics
	values AccessStats[common.Hash]

	// Previous argument-encoded operation
	prevArgOp int
}

// EventRegistryJSON is the JSON output for an event registry.
type EventRegistryJSON struct {
	Operations       []string    `json:"operations"`       // name of operations with argument classes
	StochasticMatrix [][]float64 `json:"stochasticMatrix"` // observed stochastic matrix

	// access statistics for contracts, keys, and values
	Contracts AccessStatsJSON `json:"contractStats"`
	Keys      AccessStatsJSON `json:"keyStats"`
	Values    AccessStatsJSON `json:"valueSats"`
}

// NewEventRegistry creates a new event registry.
func NewEventRegistry() EventRegistry {
	return EventRegistry{
		prevArgOp: numArgOps,
		contracts: NewAccessStats[common.Address](),
		keys:      NewAccessStats[common.Hash](),
		values:    NewAccessStats[common.Hash](),
	}
}

// RegisterOp registers an operation with no simulation arguments
func (r *EventRegistry) RegisterOp(op int) {
	if op < 0 || op >= numOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass := noArgEntry
	keyClass := noArgEntry
	valueClass := noArgEntry

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFreq(op, addrClass, keyClass, valueClass)
}

// RegisterAddressOp registers an operation with a contract-address argument
func (r *EventRegistry) RegisterAddressOp(op int, address *common.Address) {
	// check ID
	if op < 0 || op >= numOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass := r.contracts.Classify(*address)
	keyClass := noArgEntry
	valueClass := noArgEntry

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFreq(op, addrClass, keyClass, valueClass)

	// update contract address for subsequent classification
	r.contracts.Place(*address)
}

// RegisterAddressKeyOp registers an operation with a contract-address and a storage-key arguments.
func (r *EventRegistry) RegisterKeyOp(op int, address *common.Address, key *common.Hash) {
	// check operation range
	if op < 0 || op >= numOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass := r.contracts.Classify(*address)
	keyClass := r.keys.Classify(*key)
	valueClass := noArgEntry

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
	if op < 0 || op >= numOps {
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
	argOp := encodeOp(op, addr, key, value)

	// increment operation's frequency depending on argument class
	r.argOpFreq[argOp]++

	// skip counting for the first operation (there is no previous one)
	if r.prevArgOp < numArgOps {
		r.transitFreq[r.prevArgOp][argOp] = r.transitFreq[r.prevArgOp][argOp] + 1
	}
	r.prevArgOp = argOp
}

// encodeOp encodes operation and argument classes via Horner's scheme to a single value.
func encodeOp(op int, addr int, key int, value int) int {
	if op < 0 || op >= numOps {
		log.Fatalf("invalid range for operation")
	}
	if addr < 0 || addr >= numClasses {
		log.Fatalf("invalid range for contract-address")
	}
	if key < 0 || key >= numClasses {
		log.Fatalf("invalid range for storage-key")
	}
	if value < 0 || value >= numClasses {
		log.Fatalf("invalid range for storage-value")
	}
	return (((int(op)*numClasses)+addr)*numClasses+key)*numClasses + value
}

// decodeOp decodes operation with arguments.
func decodeOp(argop int) (int, int, int, int) {
	if argop < 0 || argop >= numArgOps {
		log.Fatalf("invalid range for decoding")
	}

	value := argop % numClasses
	argop = argop / numClasses

	key := argop % numClasses
	argop = argop / numClasses

	addr := argop % numClasses
	argop = argop / numClasses

	op := argop

	return op, addr, key, value
}

// opLabel produces a label for an operation with its argument classes.
func opLabel(op int, addr int, key int, value int) string {
	return fmt.Sprintf("%v%v%v%v", opText[op], classText[addr], classText[key], classText[value])
}

// NewEventRegistry produces the JSON output for an event registry.
func (r *EventRegistry) NewEventRegistryJSON() EventRegistryJSON {
	// generate labels for observable operations
	label := []string{}
	for argop := 0; argop < numArgOps; argop++ {
		if r.argOpFreq[argop] > 0 {
			// decode argument-encoded operation
			op, addr, key, value := decodeOp(argop)
			label = append(label, opLabel(op, addr, key, value))
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
	return EventRegistryJSON{
		Operations:       label,
		StochasticMatrix: A,
		Contracts:        r.contracts.NewAccessStatsJSON(),
		Keys:             r.keys.NewAccessStatsJSON(),
		Values:           r.values.NewAccessStatsJSON(),
	}
}
