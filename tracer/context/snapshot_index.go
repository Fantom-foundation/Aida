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

package context

import (
	"errors"
)

// SnapshotIndex is a translation table that maps a recorded snapshot-id to
// a replayed snapshot-id.  The recorded-id snapshot may not coincide with
// the replayed snapshot-id. Hence, we use a mapping that correlates the
// recorded/replayed snapshot-ids for reverting a snapshot when replayed.
type SnapshotIndex struct {
	recordedToReplayed map[int32]int32 // recorded/replayed snapshot map
}

// Init initializes snapshot index
func (sn *SnapshotIndex) Init() {
	sn.recordedToReplayed = make(map[int32]int32)
}

// NewSnapshotIndex creates a new snapshot index data structure.
func NewSnapshotIndex() *SnapshotIndex {
	return &SnapshotIndex{
		recordedToReplayed: make(map[int32]int32),
	}
}

// Add new snapshot-id mapping.
func (sn *SnapshotIndex) Add(recordedID int32, replayedID int32) {
	sn.recordedToReplayed[recordedID] = replayedID
}

// Get replayed snapshot-id from a recorded snapshot-id.
func (sn *SnapshotIndex) Get(recordedID int32) (int32, error) {
	replayedID, ok := sn.recordedToReplayed[recordedID]
	if !ok {
		return 0, errors.New("snapshot-id does not exist")
	}
	return replayedID, nil
}

// Size returns number of entries in the snapshot translation table.
func (sn *SnapshotIndex) Size() int {
	return len(sn.recordedToReplayed)
}
