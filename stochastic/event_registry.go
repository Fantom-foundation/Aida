package stochastic

import (
	"log"

	"github.com/ethereum/go-ethereum/common"
)

// Number of operations with encoded argument classes
const numArgEncodedOps = numStochasticOps * numClasses * numClasses * numClasses

// EventRegistry is a Facade for counting events.
type EventRegistry struct {
	// Frequency of operations which is a tensor for following map:
	//   (op,address-class,key-class,storage-class) -> frequency
	operationFrequency [numStochasticOps][numClasses][numClasses][numClasses]uint64

	// Transition counts for computing the stochastic matrix
	transitionFrequency [numArgEncodedOps][numArgEncodedOps]uint64

	// Previous operation with encoded argument classes
	prevOp int

	// Contract-address access statistics
	contractStats AccessStats[common.Address]

	// Storage Key access statistics
	keyStats AccessStats[common.Hash]

	// Storage Value access statistics
	valueStats AccessStats[common.Hash]
}

// NewEventRegistry creates a new event registry.
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		prevOp:        numArgEncodedOps,
		contractStats: NewAccessStats[common.Address](),
		keyStats:      NewAccessStats[common.Hash](),
		valueStats:    NewAccessStats[common.Hash](),
	}
}

// RegisterOp registers an operation with no simulation arguments
func (r *EventRegistry) RegisterOp(op byte) {
	if op >= numStochasticOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass, keyClass, valueClass := defaultEntry, defaultEntry, defaultEntry

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFrequency(op, addrClass, keyClass, valueClass)
}

// RegisterAddressOp registers an operation with a contract-address argument
func (r *EventRegistry) RegisterAddressOp(op byte, address *common.Address) {
	// check ID
	if op >= numStochasticOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass, keyClass, valueClass := r.classifyAddress(address), defaultEntry, defaultEntry

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFrequency(op, addrClass, keyClass, valueClass)

	// update contract address for subsequent classification
	r.putAddress(address)
}

// RegisterAddressKeyOp registers an operation with a contract-address and a storage-key arguments.
func (r *EventRegistry) RegisterKeyOp(op byte, address *common.Address, key *common.Hash) {
	// check ID
	if op >= numStochasticOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass, keyClass, valueClass := r.classifyAddress(address), r.classifyKey(key), defaultEntry

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFrequency(op, addrClass, keyClass, valueClass)

	// update contract address and storage key for subsequent classification
	r.putAddress(address)
	r.putKey(key)
}

// RegisterAddressKeyOp registers an operation with a contract-address, a storage-key and storage-value arguments.
func (r *EventRegistry) RegisterValueOp(op byte, address *common.Address, key *common.Hash, value *common.Hash) {
	// check ID
	if op >= numStochasticOps {
		log.Fatalf("invalid stochastic operation ID")
	}

	// classify simulation arguments
	addrClass, keyClass, valueClass := r.classifyAddress(address), r.classifyKey(key), r.classifyKey(value)

	// update operations's frequency and transition frequency
	// depending on type of simulation arguments
	r.updateFrequency(op, addrClass, keyClass, valueClass)

	// update contract address and storage key for subsequent classification
	r.putAddress(address)
	r.putKey(key)
	r.putValue(value)
}

// update operation and transition frequency depending on simulation argument class
func (r *EventRegistry) updateFrequency(op byte, addrClass int, keyClass int, valueClass int) {
	// increment operation's frequency depending on argument class
	r.operationFrequency[op][addrClass][keyClass][valueClass]++

	// encode argument classes to compute specialized operation using a Horner's scheme
	sOp := (((int(op)*numClasses)+addrClass)*numClasses+keyClass)*numClasses + valueClass

	// skip counting for the first operation
	if r.prevOp < numArgEncodedOps {
		r.transitionFrequency[r.prevOp][sOp]++
	}
	r.prevOp = sOp
}

// classify address based on previous addresses
func (r *EventRegistry) classifyAddress(address *common.Address) int {
	return r.contractStats.Classify(*address)
}

// putAddress puts contract address into address access-stats
func (r *EventRegistry) putAddress(address *common.Address) {
	r.contractStats.Put(*address)
}

// classifyKey storage keys based on previous keys
func (r *EventRegistry) classifyKey(key *common.Hash) int {
	return r.keyStats.Classify(*key)
}

// putKey puts storage key into key access-stats
func (r *EventRegistry) putKey(key *common.Hash) {
	r.keyStats.Put(*key)
}

// classifyValue storage value based on previous values
func (r *EventRegistry) classifyValue(value *common.Hash) int {
	return r.valueStats.Classify(*value)
}

// putValue puts storage value into value access-stats
func (r *EventRegistry) putValue(value *common.Hash) {
	r.valueStats.Put(*value)
}
