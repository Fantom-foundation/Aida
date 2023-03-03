package stochastic

import (
	"log"
	"sort"
)

// EventData contains the statistical data for events that is used for visualization.
type EventData struct {
	Contracts AccessData   // contract-address view model
	Keys      AccessData   // storage-key view model
	Values    AccessData   // storage-value view model
	Snapshot  SnapshotData // snapshot view model

	Stationary       []OpData                // stationary distribution model
	TxOperation      []OpData                // average number of operations per Tx
	TxPerBlock       float64                 // average number of transactions per block
	BlocksPerEpoch   float64                 // average number of blocks per epoch
	OperationLabel   []string                // operation labels for stochastic matrix
	StochasticMatrix [][]float64             // stochastic Matrix
	SimplifiedMatrix [numOps][numOps]float64 // simplified stochastic matrix
}

// AccessData contains the statistical data for access statistics that is used for visualization.
type AccessData struct {
	ECdf   [][2]float64 // empirical cumulative distribution function of counting stats
	QPdf   []float64    // queuing distribution function
	Lambda float64      // exponential Distribution Parameter
	Cdf    [][2]float64 // parameterised cumulative distribution function
}

// SnapshotData contains the statistical data for snapshot deltas used for visualization.
type SnapshotData struct {
	ECdf   [][2]float64 // empirical cumulative distribution function
	Lambda float64      // exponential Distribution Parameter
	Cdf    [][2]float64 // parameterised cumulative distribution function
}

// OpData stores a single operation and its probability (for stead-state)
type OpData struct {
	label string  // operation's label
	value float64 // operation's value (either probability or frequency)
}

// events is the singleton for the viewing model.
var events EventData

// GetEventsData returns the pointer to the singleton.
func GetEventsData() *EventData {
	return &events
}

// PopulateEvents populates the event model from event registry.
func (e *EventData) PopulateEventData(d *EventRegistryJSON) {

	// populate access stats for contract addresses
	e.Contracts.PopulateAccessStats(&d.Contracts)

	// populate access stats for storage keys
	e.Keys.PopulateAccessStats(&d.Keys)

	// populate access stats for storage values
	e.Values.PopulateAccessStats(&d.Values)

	// populate access stats for storage values
	e.Snapshot.PopulateSnapshotStats(d)

	// Sort entries of the stationary distribution and populate
	n := len(d.Operations)
	stationary, _ := StationaryDistribution(d.StochasticMatrix)
	data := []OpData{}
	for i := 0; i < n; i++ {
		data = append(data, OpData{
			label: d.Operations[i],
			value: stationary[i],
		})
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].value < data[j].value
	})
	e.Stationary = data

	// compute average number of operations per transaction

	// find the BeginTransaction probability in the stationary distribution
	txProb := 0.0
	blockProb := 0.0
	epochProb := 0.0
	for i := 0; i < n; i++ {
		sop, _, _, _ := DecodeOpcode(d.Operations[i])
		if sop == BeginTransactionID {
			txProb = stationary[i]
		}
		if sop == BeginBlockID {
			blockProb = stationary[i]
		}
		if sop == BeginEpochID {
			epochProb = stationary[i]
		}
	}
	if blockProb > 0.0 {
		e.TxPerBlock = txProb / blockProb
	}
	if epochProb > 0.0 {
		e.BlocksPerEpoch = blockProb / epochProb
	}

	txData := []OpData{}
	if txProb > 0.0 {
		for op := 0; op < numOps; op++ {
			// exclude scoping operations
			if op != BeginBlockID && op != EndBlockID && op != BeginEpochID && op != EndEpochID && op != BeginTransactionID && op != EndTransactionID {
				// sum all versions of an operation and normalize the value with the transaction's probability
				sum := 0.0
				for i := 0; i < n; i++ {
					if sop, _, _, _ := DecodeOpcode(d.Operations[i]); sop == op {
						sum += stationary[i]
					}
				}
				txData = append(txData, OpData{
					label: opMnemo[op],
					value: sum / txProb})
			}
		}
	}
	// sort expected operation frequencies
	sort.Slice(txData, func(i, j int) bool {
		return txData[i].value > txData[j].value
	})
	e.TxOperation = txData

	// Populate stochastic matrix
	e.OperationLabel = make([]string, len(d.Operations))
	copy(e.OperationLabel, d.Operations)
	e.StochasticMatrix = make([][]float64, len(d.StochasticMatrix))
	for i := range d.StochasticMatrix {
		e.StochasticMatrix[i] = make([]float64, len(d.StochasticMatrix[i]))
		copy(e.StochasticMatrix[i], d.StochasticMatrix[i])
	}

	// reduce stochastic matrix to a simplified matrix
	for i := 0; i < n; i++ {
		iop, _, _, _ := DecodeOpcode(d.Operations[i])
		for j := 0; j < n; j++ {
			jop, _, _, _ := DecodeOpcode(d.Operations[j])
			e.SimplifiedMatrix[iop][jop] += d.StochasticMatrix[i][j]
		}
	}

	// normalize row data after reduction
	for i := 0; i < numOps; i++ {
		sum := 0.0
		for j := 0; j < numOps; j++ {
			sum += e.SimplifiedMatrix[i][j]
		}
		for j := 0; j < numOps; j++ {
			e.SimplifiedMatrix[i][j] /= sum
		}
	}
}

// PopulateAccessStats populates access stats model
func (a *AccessData) PopulateAccessStats(d *AccessStatsJSON) {
	a.ECdf = make([][2]float64, len(d.CountingStats.ECdf))
	copy(a.ECdf, d.CountingStats.ECdf)
	lambda, err := ApproximateLambda(d.CountingStats.ECdf)
	if err != nil {
		log.Fatalf("Failed to approximate lambda parameter. Error: %v", err)
	}
	a.Lambda = lambda
	a.Cdf = PiecewiseLinearCdf(lambda, numDistributionPoints)
	a.QPdf = make([]float64, len(d.QueuingStats.Distribution))
	copy(a.QPdf, d.QueuingStats.Distribution)
}

// PopulateSnapStats populates snapshot stats model
func (s *SnapshotData) PopulateSnapshotStats(d *EventRegistryJSON) {
	s.ECdf = make([][2]float64, len(d.SnapshotEcdf))
	copy(s.ECdf, d.SnapshotEcdf)
	lambda, err := ApproximateLambda(d.SnapshotEcdf)
	if err != nil {
		log.Fatalf("Failed to approximate lambda parameter. Error: %v", err)
	}
	s.Lambda = lambda
	s.Cdf = PiecewiseLinearCdf(lambda, numDistributionPoints)
}
