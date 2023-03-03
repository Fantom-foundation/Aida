package stochastic

import (
	"log"

	"github.com/Fantom-foundation/Aida/stochastic/exponential"
	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// EstimationModelJSON is the output of the estimator in JSON format.
type EstimationModelJSON struct {
	Operations       []string    `json:"operations"`
	StochasticMatrix [][]float64 `json:"stochasticMatrix"`

	Contracts EstimationStatsJSON `json:"contractStats"`
	Keys      EstimationStatsJSON `json:"keyStats"`
	Values    EstimationStatsJSON `json:"valueStats"`

	SnapshotLambda float64 `json:"snapshotLambda"`
}

// EstimationStatsJSON is an estimated access statistics in JSON format.
type EstimationStatsJSON struct {
	NumKeys           int64     `json:"n"`
	Lambda            float64   `json:"exponentialParameter"`
	QueueDistribution []float64 `json:"queuingDistribution"`
}

// NewEstimationModelJSON creates a new estimation model.
func NewEstimationModelJSON(d *EventRegistryJSON) EstimationModelJSON {
	// copy operation codes
	operations := make([]string, len(d.Operations))
	copy(operations, d.Operations)

	// copy stochastic matrix
	stochasticMatrix := make([][]float64, len(d.StochasticMatrix))
	for i := range d.StochasticMatrix {
		stochasticMatrix[i] = make([]float64, len(d.StochasticMatrix[i]))
		copy(stochasticMatrix[i], d.StochasticMatrix[i])
	}

	// compute snapshot lambda
	snapshotLambda, err := exponential.ApproximateLambda(d.SnapshotEcdf)
	if err != nil {
		log.Fatalf("Failed to approximate lambda parameter. Error: %v", err)
	}

	return EstimationModelJSON{
		Operations:       operations,
		StochasticMatrix: stochasticMatrix,
		Contracts:        NewEstimationStats(&d.Contracts),
		Keys:             NewEstimationStats(&d.Keys),
		Values:           NewEstimationStats(&d.Values),
		SnapshotLambda:   snapshotLambda,
	}
}

// NewEstimationStats creates a new EstimationStatsJSON objects for an access statistics.
func NewEstimationStats(d *statistics.AccessJSON) EstimationStatsJSON {
	// compute lambda
	lambda, err := exponential.ApproximateLambda(d.Counting.ECdf)
	if err != nil {
		log.Fatalf("Failed to approximate lambda parameter. Error: %v", err)
	}

	// copy queuing distribution
	distribution := make([]float64, len(d.Queuing.Distribution))
	copy(distribution, d.Queuing.Distribution)

	return EstimationStatsJSON{
		Lambda:            lambda,
		NumKeys:           d.Counting.NumKeys,
		QueueDistribution: distribution,
	}
}
