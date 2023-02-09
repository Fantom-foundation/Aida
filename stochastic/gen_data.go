package stochastic

import (
	"log"
	"sort"
)

// EventData contains the statistical data for events that is used for visualization.
type EventData struct {
	Contracts AccessData // contract-address view model
	Keys      AccessData // storage-key view model
	Values    AccessData // storage-value view model

	SteadyState      []OpData    // steady-state model
	OperationLabel   []string    // operation labels for stochastic matrix
	StochasticMatrix [][]float64 // stochastic Matrix
}

// AccessData contains the statistical data for access statistics that is used for visualization.
type AccessData struct {
	ECdf   [][2]float64 // empirical cumulative distribution function of counting stats
	QPdf   []float64    // queuing distribution function
	Lambda float64      // exponential Distribution Parameter
	Cdf    [][2]float64 // parameterised cumulative distribution function
}

// OpData stores a single operation and its probability (for stead-state)
type OpData struct {
	label string  // operation's label
	p     float64 // operation's probability
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

	// Sort entries of the steady and populate
	n := len(d.Operations)
	steadyState := SteadyStateDistribution(d.StochasticMatrix)
	data := []OpData{}
	for i := 0; i < n; i++ {
		data = append(data, OpData{
			label: d.Operations[i],
			p:     steadyState[i],
		})
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].p < data[j].p
	})
	e.SteadyState = data

	// Populate stochastic matrix
	e.OperationLabel = make([]string, len(d.Operations))
	copy(e.OperationLabel, d.Operations)
	e.StochasticMatrix = make([][]float64, len(d.StochasticMatrix))
	for i := range d.StochasticMatrix {
		e.StochasticMatrix[i] = make([]float64, len(d.StochasticMatrix[i]))
		copy(e.StochasticMatrix[i], d.StochasticMatrix[i])
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
