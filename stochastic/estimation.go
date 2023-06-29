package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/stochastic/exponential"
	"github.com/Fantom-foundation/Aida/stochastic/statistics"
)

// EstimationModelJSON is the output of the estimator in JSON format.
type EstimationModelJSON struct {
	FileId           string      `json:"FileId"`
	Operations       []string    `json:"operations"`
	StochasticMatrix [][]float64 `json:"stochasticMatrix"`

	Contracts EstimationStatsJSON `json:"contractStats"`
	Keys      EstimationStatsJSON `json:"keyStats"`
	Values    EstimationStatsJSON `json:"valueStats"`

	SnapshotLambda float64 `json:"snapshotLambda"`
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
		log.Fatalf("failed to approximate lambda parameter; %v", err)
	}

	// construct JSON object for simulation
	return EstimationModelJSON{
		FileId:           "simulation",
		Operations:       operations,
		StochasticMatrix: stochasticMatrix,
		Contracts:        NewEstimationStats(&d.Contracts),
		Keys:             NewEstimationStats(&d.Keys),
		Values:           NewEstimationStats(&d.Values),
		SnapshotLambda:   snapshotLambda,
	}
}

// ReadSimulation reads the simulation file in JSON format (generated by the estimator).
func ReadSimulation(filename string) (*EstimationModelJSON, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed opening simulation file; %v", err)
	}
	defer file.Close()
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed reading simulation file; %v", err)
	}
	var simulation EstimationModelJSON
	err = json.Unmarshal(contents, &simulation)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling JSON; %v", err)
	}
	if simulation.FileId != "simulation" {
		return nil, fmt.Errorf("file %v is not a simulation file", filename)
	}
	return &simulation, nil
}

// WriteJSON writes an simulation file in JSON format.
func (m *EstimationModelJSON) WriteJSON(filename string) error {
	f, fErr := os.Create(filename)
	if fErr != nil {
		return fmt.Errorf("cannot open JSON file; %v", fErr)
	}
	defer f.Close()
	jOut, jErr := json.MarshalIndent(m, "", "    ")
	if jErr != nil {
		return fmt.Errorf("failed to convert JSON file; %v", jErr)
	}
	_, pErr := fmt.Fprintln(f, string(jOut))
	if pErr != nil {
		return fmt.Errorf("failed to convert JSON file; %v", pErr)
	}
	return nil
}

// EstimationStatsJSON is an estimated access statistics in JSON format.
type EstimationStatsJSON struct {
	NumKeys           int64     `json:"n"`
	Lambda            float64   `json:"exponentialParameter"`
	QueueDistribution []float64 `json:"queuingDistribution"`
}

// NewEstimationStats creates a new EstimationStatsJSON objects for an access statistics.
func NewEstimationStats(d *statistics.AccessJSON) EstimationStatsJSON {
	// compute lambda
	lambda, err := exponential.ApproximateLambda(d.Counting.ECdf)
	if err != nil {
		log.Fatalf("failed to approximate lambda parameter; %v", err)
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
