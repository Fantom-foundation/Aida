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
