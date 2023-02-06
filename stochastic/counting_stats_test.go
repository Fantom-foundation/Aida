package stochastic

import (
	"encoding/json"
	"testing"
)

// TestCountingStatsSimple1 counts a single occurrence of an item and checks whether
// its freq is one.
func TestCountingStatsSimple1(t *testing.T) {
	stats := NewCountingStats[int]()
	stats.Count(100)
	freq := stats.Frequency(100)
	if freq != 1 {
		t.Fatalf("Counting failed")
	}
	if !stats.Exists(100) {
		t.Fatalf("Counting failed")
	}
}

// TestCountingStatsSimple2 counts two occurrences of a data item and checks whether its
// freq is two.
func TestCountingStatsSimple2(t *testing.T) {
	stats := NewCountingStats[int]()
	data := 200
	stats.Count(data)
	stats.Count(data)
	freq := stats.Frequency(data)
	if freq != 2 {
		t.Fatalf("Counting failed")
	}
}

// TestCountingStatsSimple3 counts the single occurrence of two items and checks whether
// their frequencies are one and whether they exist.
func TestCountingStatsSimple3(t *testing.T) {
	stats := NewCountingStats[int]()
	data1 := 10
	data2 := 11
	stats.Count(data1)
	stats.Count(data2)
	freq1 := stats.Frequency(data1)
	freq2 := stats.Frequency(data2)
	if freq1 != 1 || freq2 != 1 || !stats.Exists(data1) || !stats.Exists(data2) {
		t.Fatalf("Counting failed failed")
	}
}

// TestCountingStatsSimple4 tests JSON output of distribution.
func TestCountingStatsSimple4(t *testing.T) {
	stats := NewCountingStats[int]()
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
		stats.Count(i)
	}
	stats.Count(1)
	stats.Count(10)

	// produce distribution in JSON format
	// Case 1: number entries are smaller than observerd number of items.
	jOut, err = json.Marshal(stats.produceJSON(4))
	if err != nil {
		t.Fatalf("Marshalling failed to produce distribution")
	}
	expected = `{"n":10,"ecdf":[[0,0],[0.15,0.3333333333333333],[0.35,0.5],[0.55,0.6666666666666666],[0.75,0.8333333333333333],[0.95,1],[1,1]]}`
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
