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

// IDs for argument classes
const (
	NoArgID         = iota // default label (for no argument)
	ZeroValueID            // zero value access
	NewValueID             // newly occurring value access
	PreviousValueID        // value that was previously accessed
	RecentValueID          // value that recently accessed (time-window is fixed to statistics.QueueLen)
	RandomValueID          // random access (everything else)

	NumClasses
)

// number of points on the ecdf
const NumDistributionPoints = 100

// QueueLen sets the length of queuing statistics.
// NB: must be greater than one.
const QueueLen = 32
