// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package stochastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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

// NewEventRegistry creates a new event registry.
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
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

	// update operation's frequency and transition frequency
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

	// update operations' frequency and transition frequency
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

	// update operation's frequency and transition frequency
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

// RegisterSnapshotDelta counts the delta of a snapshot. The delta is
// the number of stack elements between the top-of-stack of the snapshot stack
// and the reverted snapshot. A delta of zero means that the state is reverted
// to the previously created snapshot, a delta of one means that the state is
// reverted to the previous of the previous snapshot and so on.
func (r *EventRegistry) RegisterSnapshotDelta(delta int) {
	r.snapshotFreq[delta]++
}

// WriteJSON writes an event registry in JSON format.
func (r *EventRegistry) WriteJSON(filename string) error {
	f, fErr := os.Create(filename)
	if fErr != nil {
		return fmt.Errorf("cannot open JSON file; %v", fErr)
	}
	defer f.Close()
	jOut, jErr := json.MarshalIndent(r.NewEventRegistryJSON(), "", "    ")
	if jErr != nil {
		return fmt.Errorf("failed to convert JSON file; %v", jErr)
	}
	_, pErr := fmt.Fprintln(f, string(jOut))
	if pErr != nil {
		return fmt.Errorf("failed to convert JSON file; %v", pErr)
	}
	return nil
}

// EventRegistryJSON is the JSON struct for an event registry.
type EventRegistryJSON struct {
	FileId           string      `json:"FileId"`           // file identification
	Operations       []string    `json:"operations"`       // name of operations with argument classes
	StochasticMatrix [][]float64 `json:"stochasticMatrix"` // observed stochastic matrix

	// access statistics for contracts, keys, and values
	Contracts statistics.AccessJSON `json:"contractStats"`
	Keys      statistics.AccessJSON `json:"keyStats"`
	Values    statistics.AccessJSON `json:"valueSats"`

	// snapshot delta frequencies
	SnapshotEcdf [][2]float64 `json:"snapshotEcdf"`
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
		FileId:           "events",
		Operations:       label,
		StochasticMatrix: A,
		Contracts:        r.contracts.NewAccessJSON(),
		Keys:             r.keys.NewAccessJSON(),
		Values:           r.values.NewAccessJSON(),
		SnapshotEcdf:     eCdf,
	}
}

// ReadEventsJSON reads event file in JSON format.
func ReadEvents(filename string) (*EventRegistryJSON, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed opening event file %v; %v", filename, err)
	}
	defer file.Close()
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed reading event file; %v", err)
	}
	var eventsJSON EventRegistryJSON
	err = json.Unmarshal(contents, &eventsJSON)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal event registry; %v", err)
	}
	if eventsJSON.FileId != "events" {
		return nil, fmt.Errorf("file %v is not an events file", filename)
	}
	return &eventsJSON, nil
}
