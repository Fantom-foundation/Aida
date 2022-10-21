package dict

import (
	"testing"
)

// Add()
// Positive Test: Add a new set of mappings and compare the size of index map
func TestPositiveSnapshotIndexAdd(t *testing.T) {
	var recordedID1 uint16 = 1
	var recordedID2 uint16 = 2
	var replayedID1 uint16 = 0
	var replayedID2 uint16 = 1
	snapshotIdx := NewSnapshotIndex()
	snapshotIdx.Add(recordedID1, replayedID1)
	snapshotIdx.Add(recordedID2, replayedID2)

	want := 2
	have := len(snapshotIdx.recordedToReplayed)
	if have != want {
		t.Fatalf("Unexpected map size")
	}
}

// Positive Test: Add an ID twice, and check index result.
func TestPositiveSnapshotIndexAddDuplicateID(t *testing.T) {
	var recordedID uint16 = 1
	var replayedID uint16 = 0
	snapshotIdx := NewSnapshotIndex()
	snapshotIdx.Add(recordedID, replayedID)
	replayedID = 2
	snapshotIdx.Add(recordedID, replayedID)
	want := 1
	have := len(snapshotIdx.recordedToReplayed)
	if have != want {
		t.Fatalf("Unexpected map size")
	}

	ID, _ := snapshotIdx.Get(recordedID)
	if ID != replayedID {
		t.Fatalf("Unexpected replayed snapshot index")
	}
}

// Get()
// Positive Test: Add ID to SnapshotIndex and compare with index result.
func TestPositiveSnapshotIndexGet(t *testing.T) {
	var recordedID uint16 = 1
	var replayedID uint16 = 8
	snapshotIdx := NewSnapshotIndex()
	snapshotIdx.Add(recordedID, replayedID)
	ID, err := snapshotIdx.Get(recordedID)
	if err != nil {
		t.Fatalf("Failed to get snapshot-id %v. Error: %v", recordedID, err)
	}
	if replayedID != ID {
		t.Fatalf("ID mismatched")
	}
}

// Negative Test: ID of Get mismatches.
func TestNegativeSnapshotIndexGet(t *testing.T) {
	var recordedID uint16 = 1
	var replayedID uint16 = 8
	snapshotIdx := NewSnapshotIndex()
	snapshotIdx.Add(recordedID, replayedID)
	_, err := snapshotIdx.Get(recordedID + 1)
	if err == nil {
		t.Fatalf("Failed to report error. ID %v doesn't exist", recordedID+1)
	}
}
