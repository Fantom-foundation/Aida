package dictionary

import (
	"testing"
)

// TestSnapshotIndexAdd adds a new set of mappings and compares the size of index map.
func TestSnapshotIndexAdd(t *testing.T) {
	var recordedID1 int32 = 1
	var recordedID2 int32 = 2
	var replayedID1 int32 = 0
	var replayedID2 int32 = 1
	snapshotIdx := NewSnapshotIndex()
	snapshotIdx.Add(recordedID1, replayedID1)
	snapshotIdx.Add(recordedID2, replayedID2)
	want := 2
	have := len(snapshotIdx.recordedToReplayed)
	if have != want {
		t.Fatalf("Unexpected map size")
	}
}

// TestSnapshotIndexAddDuplicateID adds an ID twice, and checks index result.
func TestSnapshotIndexAddDuplicateID(t *testing.T) {
	var recordedID int32 = 1
	var replayedID int32 = 0
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

// TestSnapshotIndexGet adds ID to SnapshotIndex and compares with index result.
func TestSnapshotIndexGet1(t *testing.T) {
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

// TestSnapshotIndexGet checks ID of Get mismatches.
func TestSnapshotIndexGet2(t *testing.T) {
	var recordedID int32 = 1
	var replayedID int32 = 8
	snapshotIdx := NewSnapshotIndex()
	snapshotIdx.Add(recordedID, replayedID)
	_, err := snapshotIdx.Get(recordedID + 1)
	if err == nil {
		t.Fatalf("Failed to report error. ID %v doesn't exist", recordedID+1)
	}
}
