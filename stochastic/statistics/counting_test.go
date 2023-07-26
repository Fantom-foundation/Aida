package statistics

import (
	"encoding/json"
	"testing"
)

// TestCountingSimple1 counts a single occurrence of an item and checks whether
// its freq is one.
func TestCountingSimple1(t *testing.T) {
	stats := NewCounting[int]()
	if stats.Exists(100) {
		t.Fatalf("Counting failed")
	}
	stats.Place(100)
	if !stats.Exists(100) {
		t.Fatalf("Counting failed")
	}
}

// TestCountingSimple2 counts two occurrences of a data item and checks whether its
// freq is two.
func TestCountingSimple2(t *testing.T) {
	stats := NewCounting[int]()
	data := 200
	if stats.Exists(data) {
		t.Fatalf("Counting failed")
	}
	stats.Place(data)
	stats.Place(data)
	if !stats.Exists(data) {
		t.Fatalf("Counting failed")
	}
}

// TestCountingSimple3 counts the single occurrence of two items and checks whether
// their frequencies are one and whether they exist.
func TestCountingSimple3(t *testing.T) {
	stats := NewCounting[int]()
	data1 := 10
	data2 := 11
	stats.Place(data1)
	stats.Place(data2)
	if !stats.Exists(data1) || !stats.Exists(data2) {
		t.Fatalf("Counting failed failed")
	}
}

// TestCountingSimple4 tests JSON output of distribution.
func TestCountingSimple4(t *testing.T) {
	stats := NewCounting[int]()
	// produce distribution in JSON format
	// Case 0: number entries are smaller than observerd number of items.
	jOut, err := json.Marshal(stats.produceJSON(4))
	if err != nil {
		t.Fatalf("Marshalling failed to produce distribution")
	}
	expected := `{"n":0,"ecdf":[]}`
	if string(jOut) != expected {
		t.Fatalf("case 0: produced wrong JSON output (%v)", string(jOut))
	}

	for i := 1; i <= 10; i++ {
		stats.Place(i)
	}
	stats.Place(1)
	stats.Place(10)

	// produce distribution in JSON format
	// Case 1: number entries are smaller than observerd number of items.
	jOut, err = json.Marshal(stats.produceJSON(4))
	if err != nil {
		t.Fatalf("Marshalling failed to produce distribution")
	}
	expected = `{"n":10,"ecdf":[[0,0],[0.15,0.3333333333333333],[0.95,1],[1,1]]}`
	if string(jOut) != expected {
		t.Fatalf("case 1: produced wrong JSON output (%v)", string(jOut))
	}

	// produce distribution in JSON format
	// Case 2: number entries are greater than observerd number of items.
	jOut, err = json.Marshal(stats.produceJSON(100))
	if err != nil {
		t.Fatalf("Marshalling failed to produce distribution")
	}
	expected = `{"n":10,"ecdf":[[0,0],[0.05,0.16666666666666666],[0.15,0.3333333333333333],[0.25,0.41666666666666663],[0.35,0.5],[0.45,0.5833333333333333],[0.55,0.6666666666666666],[0.65,0.75],[0.75,0.8333333333333333],[0.85,0.9166666666666666],[0.95,1],[1,1]]}`
	if string(jOut) != expected {
		t.Fatalf("case 2: produced wrong JSON output (%v)", string(jOut))
	}
}
