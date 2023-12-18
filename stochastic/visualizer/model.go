package visualizer

import (
	"log"
	"sort"

	"github.com/Fantom-foundation/Aida/stochastic"
	"github.com/Fantom-foundation/Aida/stochastic/exponential"
	"github.com/Fantom-foundation/Aida/stochastic/stationary"
	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// EventData contains the statistical data for events that is used for visualization.
type EventData struct {
	Contracts AccessData   // contract-address view model
	Keys      AccessData   // storage-key view model
	Values    AccessData   // storage-value view model
	Snapshot  SnapshotData // snapshot view model

	Stationary          []OpData                                      // stationary distribution model
	TxOperation         []OpData                                      // average number of operations per Tx
	TxPerBlock          float64                                       // average number of transactions per block
	BlocksPerSyncPeriod float64                                       // average number of blocks per sync-period
	OperationLabel      []string                                      // operation labels for stochastic matrix
	StochasticMatrix    [][]float64                                   // stochastic Matrix
	SimplifiedMatrix    [stochastic.NumOps][stochastic.NumOps]float64 // simplified stochastic matrix
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
func (e *EventData) PopulateEventData(d *stochastic.EventRegistryJSON) {

	// populate access stats for contract addresses
	e.Contracts.PopulateAccess(&d.Contracts)

	// populate access stats for storage keys
	e.Keys.PopulateAccess(&d.Keys)

	// populate access stats for storage values
	e.Values.PopulateAccess(&d.Values)

	// populate access stats for storage values
	e.Snapshot.PopulateSnapshotStats(d)

	// Sort entries of the stationary distribution and populate
	n := len(d.Operations)
	stationary, _ := stationary.ComputeDistribution(d.StochasticMatrix)
	opData := []OpData{}
	for i := 0; i < n; i++ {
		opData = append(opData, OpData{
			label: d.Operations[i],
			value: stationary[i],
		})
	}
	sort.Slice(opData, func(i, j int) bool {
		return opData[i].value < opData[j].value
	})
	e.Stationary = opData

	// compute average number of operations per transaction

	// find the BeginTransaction probability in the stationary distribution
	txProb := 0.0
	blockProb := 0.0
	syncPeriodProb := 0.0
	for i := 0; i < n; i++ {
		data := stochastic.DecodeOpcode(d.Operations[i])
		if data.Operation == stochastic.BeginTransactionID {
			txProb = stationary[i]
		}
		if data.Operation == stochastic.BeginBlockID {
			blockProb = stationary[i]
		}
		if data.Operation == stochastic.BeginSyncPeriodID {
			syncPeriodProb = stationary[i]
		}
	}
	if blockProb > 0.0 {
		e.TxPerBlock = txProb / blockProb
	}
	if syncPeriodProb > 0.0 {
		e.BlocksPerSyncPeriod = blockProb / syncPeriodProb
	}

	txData := []OpData{}
	if txProb > 0.0 {
		for op := 0; op < stochastic.NumOps; op++ {
			// exclude scoping operations
			if op != stochastic.BeginBlockID && op != stochastic.EndBlockID && op != stochastic.BeginSyncPeriodID && op != stochastic.EndSyncPeriodID && op != stochastic.BeginTransactionID && op != stochastic.EndTransactionID {
				// sum all versions of an operation and normalize the value with the transaction's probability
				sum := 0.0
				for i := 0; i < n; i++ {
					if data := stochastic.DecodeOpcode(d.Operations[i]); data.Operation == op {
						sum += stationary[i]
					}
				}
				txData = append(txData, OpData{
					label: stochastic.OpMnemo(op),
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
		iData := stochastic.DecodeOpcode(d.Operations[i])
		for j := 0; j < n; j++ {
			jData := stochastic.DecodeOpcode(d.Operations[j])
			e.SimplifiedMatrix[iData.Operation][jData.Operation] += d.StochasticMatrix[i][j]
		}
	}

	// normalize row opData after reduction
	for i := 0; i < stochastic.NumOps; i++ {
		sum := 0.0
		for j := 0; j < stochastic.NumOps; j++ {
			sum += e.SimplifiedMatrix[i][j]
		}
		for j := 0; j < stochastic.NumOps; j++ {
			e.SimplifiedMatrix[i][j] /= sum
		}
	}
}

// PopulateAccess populates access stats model
func (a *AccessData) PopulateAccess(d *statistics.AccessJSON) {
	a.ECdf = make([][2]float64, len(d.Counting.ECdf))
	copy(a.ECdf, d.Counting.ECdf)
	lambda, err := exponential.ApproximateLambda(d.Counting.ECdf)
	if err != nil {
		log.Fatalf("Failed to approximate lambda parameter. Error: %v", err)
	}
	a.Lambda = lambda
	a.Cdf = exponential.PiecewiseLinearCdf(lambda, statistics.NumDistributionPoints)
	a.QPdf = make([]float64, len(d.Queuing.Distribution))
	copy(a.QPdf, d.Queuing.Distribution)
}

// PopulateSnapStats populates snapshot stats model
func (s *SnapshotData) PopulateSnapshotStats(d *stochastic.EventRegistryJSON) {
	s.ECdf = make([][2]float64, len(d.SnapshotEcdf))
	copy(s.ECdf, d.SnapshotEcdf)
	lambda, err := exponential.ApproximateLambda(d.SnapshotEcdf)
	if err != nil {
		log.Fatalf("Failed to approximate lambda parameter. Error: %v", err)
	}
	s.Lambda = lambda
	s.Cdf = exponential.PiecewiseLinearCdf(lambda, statistics.NumDistributionPoints)
}
