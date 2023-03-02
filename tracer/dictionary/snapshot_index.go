package dictionary

import (
	"errors"
)

// SnapshotIndex maps a recorded snapshot-id to replayed snapshot-id.
// The recorded-id snapshot may not coincide with the replayed snapshot-id.
// Hence, we use a mapping that correlates the recorded/replayed snapshot-ids
// for reverting a snapshot.

type SnapshotIndex struct {
	recordedToReplayed map[int32]int32 // recorded/replayed snapshot map
}

// Init initializes a snapshot index.
func (oIdx *SnapshotIndex) Init() {
	oIdx.recordedToReplayed = make(map[int32]int32)
}

// NewSnapshotIndex creates a new snapshot index data structure.
func NewSnapshotIndex() *SnapshotIndex {
	p := new(SnapshotIndex)
	p.Init()
	return p
}

// Add new snapshot-id mapping.
func (oIdx *SnapshotIndex) Add(recordedID int32, replayedID int32) {
	oIdx.recordedToReplayed[recordedID] = replayedID
}

// Get replayed snapshot-id from a recorded snapshot-id.
func (oIdx *SnapshotIndex) Get(recordedID int32) (int32, error) {
	replayedID, ok := oIdx.recordedToReplayed[recordedID]
	if !ok {
		return 0, errors.New("snapshot-id does not exist")
	}
	return replayedID, nil
}

func (oIdx *SnapshotIndex) Size() int {
	return len(oIdx.recordedToReplayed)
}
