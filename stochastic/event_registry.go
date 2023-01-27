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

// Event distribution for JSON.
type EventDistribution struct {
	OperationLabel [] string             // name of operations with argument classes
	OperationDistribution [] float64 // empirical probability of an operation
	StochasticMatrix [][]float64 // observed stochastic matrix 
	ContractDistribution AccessDistribution 
	KeyDistribution AccessDistribution
	ValueDistribution AccessDistribution
}

// NewEventRegistry creates a new event registry.
func NewEventRegistry() EventRegistry {
	return EventRegistry{
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

// encodeOp encodes operation and argument classes via Horner's scheme to a single value
func encodeOp(op int, addrClass int, keyClass int, valueClass int) int {
	return (((int(op)*numClasses)+addrClass)*numClasses+keyClass)*numClasses + valueClass
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


func (r *EventRegistry) ProduceDistribution() EventDistribution {

	// compute operation distribution and its labels
	opDistribution := make([]float64, numArgEncodedOps)
	opLabel := make([]string, numArgEncodedOps)
	total := uint64(0)
	for op:=0;op<numStochasticOps;op++ { 
		for addrClass:=0;addrClass<numClasses;addrClass++ { 
			for keyClass:=0;keyClass<numClasses;keyClass++ { 
				for valueClass:=0;valueClass<numClasses;valueClass++ { 
					// retrieve frequencies for distribution
					sOp := encodeOp(op, addrClass, keyClass, valueClass)
					freq := r.operationFrequency[op][addrClass][keyClass][valueClass]
					opDistribution[sOp] = float64(freq)
					total += freq

					// generate label of operation
					label := operationText[op]+"("
					switch operationNumArgs[op] {
					case 1:
						label += classText[addrClass]
					case 2: 
						label += classText[addrClass] + "," + classText[keyClass]
					case 3:
						label += classText[addrClass] + "," + classText[keyClass] + "," + classText[valueClass]
					}
					label +=")"
					opLabel[op] = label
				}
			}
		}
	}
	for op:=0;op<numArgEncodedOps;op++ { 
		// normalize distribution
		opDistribution[op] = opDistribution[op]/float64(total)
	}

	// Compute stochastic matrix
	stochasticMatrix := make([][]float64, numArgEncodedOps)
	for i:=0;i<numArgEncodedOps;i++ { 
		stochasticMatrix[i] = make([]float64, numArgEncodedOps) 
		total := uint64(0)
		for j:=0;j<numArgEncodedOps;j++ { 
			total += r.transitionFrequency[i][j]
		}
		for j:=0;j<numArgEncodedOps;j++ { 
			stochasticMatrix[i][j] = float64(r.transitionFrequency[i][j])/float64(total)
		}
	}
	return  EventDistribution {
		OperationLabel: opLabel,
		OperationDistribution: opDistribution,
		StochasticMatrix: stochasticMatrix,
		ContractDistribution:  r.contractStats.ProduceDistribution(),
		KeyDistribution: r.keyStats.ProduceDistribution(),
		ValueDistribution: r.valueStats.ProduceDistribution(),
	}
}


