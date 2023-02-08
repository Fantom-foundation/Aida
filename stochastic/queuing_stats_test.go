package stochastic

import (
	"encoding/json"
	"testing"
)

// TestQueuingStatsSimple tests for existence/non-existence of elements.
func TestQueuingStatsSimple(t *testing.T) {
	// create index queue
	queue := NewQueuingStats[int]()

	// place first element
	queue.Place(0)

	// find first element
	pos := queue.Find(0)
	if pos != 0 {
		t.Fatalf("element cannot be found")
	}

	// unknown element must not be found
	pos = queue.Find(1)
	if pos != -1 {
		t.Fatalf("element must not be found")
	}
}

// TestQueuingStatsSimple1 tests for existence/non-existence of elements.
func TestQueuingStatsSimple1(t *testing.T) {
	// create index queue
	queue := NewQueuingStats[int]()

	// find first element
	pos := queue.Find(0)
	if pos != -1 {
		t.Fatalf("Queue must be empty")
	}

	// place first element
	queue.Place(0)

	// place second element
	queue.Place(1)

	// find first element
	pos = queue.Find(1)
	if pos != 0 {
		t.Fatalf("first element cannot be found")
	}
	pos = queue.Find(0)
	if pos != 1 {
		t.Fatalf("second element cannot be found")
	}
}

// TestQueuingStatsSimple2 tests for existence/non-existence of elements.
func TestQueuingStatsSimple2(t *testing.T) {
	// create index queue
	queue := NewQueuingStats[int]()

	// place first element
	for i := 0; i < qstatsLen+1; i++ {
		queue.Place(i)
	}

	// find first element
	pos := queue.Find(0)
	if pos != -1 {
		t.Fatalf("first element must not be found")
	}
	pos = queue.Find(1)
	if pos != qstatsLen-1 {
		t.Fatalf("second element must be found: %v", pos)
	}
	pos = queue.Find(qstatsLen)
	if pos != 0 {
		t.Fatalf("last element must be found")
	}

	queue.Place(qstatsLen + 1)

	pos = queue.Find(1)
	if pos != -1 {
		t.Fatalf("second element must not be found")
	}
	pos = queue.Find(2)
	if pos != qstatsLen-1 {
		t.Fatalf("third element must be found: %v", pos)
	}
	pos = queue.Find(qstatsLen + 1)
	if pos != 0 {
		t.Fatalf("last element must be found")
	}
}

// TestQueuingStatsJSON tests JSON output of distribution.
func TestQueuingStatsJSON(t *testing.T) {
	// create index queue
	queue := NewQueuingStats[int]()

	// place first element
	for i := 0; i < 300; i++ {
		queue.Place(i)
		// find first element
		pos := queue.Find(i)
		if pos != 0 {
			t.Fatalf("first element must be found")
		}
		pos = queue.Find(i - 1)
		pos = queue.Find(i - 2)
		pos = queue.Find(i - 3)
	}

	// produce distribution in JSON format
	jOut, err := json.Marshal(queue.NewQueuingStatsJSON())
	if err != nil {
		t.Fatalf("Marshalling failed to produce distribution")
	}
	expected := `{"distribution":[0.25125628140703515,0.25041876046901174,0.24958123953098826,0.24874371859296482,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}`
	if string(jOut) != expected {
		t.Fatalf("produced wrong JSON output %v", string(jOut))
	}
}
