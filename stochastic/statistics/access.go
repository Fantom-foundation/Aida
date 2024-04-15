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

package statistics

// Access for tracking access classes
type Access[T comparable] struct {
	// counting statistics for data accesses
	cstats Counting[T]

	// queuing statistics for data accesses
	qstats Queuing[T]
}

// AccessJSON is the JSON output for Access.
type AccessJSON struct {
	Counting CountingJSON
	Queuing  QueuingJSON
}

// NewAccess creates a new access.
func NewAccess[T comparable]() Access[T] {
	return Access[T]{NewCounting[T](), NewQueuing[T]()}
}

// Places an access into the access statistics.
func (a *Access[T]) Place(data T) {
	// don't place zero constant into queue/counting stats
	var zeroValue T
	if data == zeroValue {
		return
	}

	// Update counting statistics only if not found in queue
	if a.qstats.Find(data) == -1 {
		a.cstats.Place(data)
	}

	// Place data into queuing statistics
	a.qstats.Place(data)
}

// Classify an access depending on previous placements.
func (a *Access[T]) Classify(data T) int {
	// check zero value
	var zeroValue T
	if data == zeroValue {
		return ZeroValueID
	}
	switch a.qstats.Find(data) {
	case -1:
		// data not found in the queuing statistics
		// => check counting statistics
		if !a.cstats.Exists(data) {
			return NewValueID
		} else {
			return RandomValueID
		}
	case 0:
		// previous entry
		return PreviousValueID
	default:
		// data found in queuing statistics
		// but not previously accessed
		return RecentValueID
	}
}

// NewAccessJSON produces JSON output for an access statistics.
func (a *Access[T]) NewAccessJSON() AccessJSON {
	return AccessJSON{a.cstats.NewCountingJSON(), a.qstats.NewQueuingJSON()}
}
