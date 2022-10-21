package dict

import (
	"errors"
)

// SnapshotIndex maps a recorded snapshot-id to replayed snapshot-id.
// The recorded-id snapshot may not coincide with the replayed snapshot-id.
// Hence, we use a mapping that correlates the recorded/replayed snapshot-ids
// for reverting a snapshot.

type SnapshotIndex struct {
	recordedToReplayed map[uint16]uint16
}

// Initialize a snapshot index.
func (oIdx *SnapshotIndex) Init() {
	oIdx.recordedToReplayed = make(map[uint16]uint16)
}

// Create new snapshot index data structure.
func NewSnapshotIndex() *SnapshotIndex {
	p := new(SnapshotIndex)
	p.Init()
	return p
}

// Add new snapshot-id mapping.
func (oIdx *SnapshotIndex) Add(recordedID uint16, replayedID uint16) {
	oIdx.recordedToReplayed[recordedID] = replayedID
}

// Retrieve replayed snapshot-id from a recorded-id.
func (oIdx *SnapshotIndex) Get(recordedID uint16) (uint16, error) {
	replayedID, ok := oIdx.recordedToReplayed[recordedID]
	if !ok {
		return 0, errors.New("snapshot-id does not exist")
	}
	return replayedID, nil
}
