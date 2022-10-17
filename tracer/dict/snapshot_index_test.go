package dict

import (
	"testing"
)

// Add()
// Positive Test: Add a new set of mappings and compare the size of index map
func TestPositiveSnapshotIndexAdd(t *testing.T) {
	var recordedID1 int32 = 1
	var recordedID2 int32 = 2
	var replayedID1 int32 = 0
	var replayedID2 int32 = 1
	snapshotIdx := NewSnapshotIndex()
	err1 := snapshotIdx.Add(recordedID1, replayedID1)
	if err1 != nil {
		t.Fatalf("Failed to add new ID: %v", err1)
	}
	err2 := snapshotIdx.Add(recordedID2, replayedID2)
	if err2 != nil {
		t.Fatalf("Failed to add new ID: %v", err2)
	}
	want := 2
	have := len(snapshotIdx.recordedToReplayed)
	if have != want {
		t.Fatalf("Unexpected map size")
	}
}

// Negative Test: Add an ID twice, and check for failure.
func TestNegativeSnapshotIndexAdd(t *testing.T) {
	var recordedID int32 = 1
	var replayedID int32 = 0
	snapshotIdx := NewSnapshotIndex()
	err1 := snapshotIdx.Add(recordedID, replayedID)
	if err1 != nil {
		t.Fatalf("Failed to add mapping. Error: %v", err1)
	}
	err2 := snapshotIdx.Add(recordedID, replayedID)
	if err2 == nil {
		t.Fatalf("Expected an error when adding same mapping")
	}
	want := 1
	have := len(snapshotIdx.recordedToReplayed)
	if have != want {
		t.Fatalf("Unexpected map size")
	}
}

// Get()
// Positive Test: Add ID to SnapshotIndex and compare with index result.
func TestPositiveSnapshotIndexGet(t *testing.T) {
	var recordedID int32 = 1
	var replayedID int32 = 8
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
	var recordedID int32 = 1
	var replayedID int32 = 8
	snapshotIdx := NewSnapshotIndex()
	snapshotIdx.Add(recordedID, replayedID)
	_, err := snapshotIdx.Get(recordedID + 1)
	if err == nil {
		t.Fatalf("Failed to report error. ID %v doesn't exist", recordedID+1)
	}
}
