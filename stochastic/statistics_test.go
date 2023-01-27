package stochastic

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

// TestStatisticsSimple1 counts a single occurrence of an address and checks whether
// its frequency is one.
func TestStatisticsSimple1(t *testing.T) {
	stats := NewStatistics[common.Address]()
	address := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	stats.Count(address)
	frequency := stats.Frequency(address)
	if frequency != 1 {
		t.Fatalf("Counting failed")
	}
}

// TestStatisticsSimple2 counts two occurrences of an address and checks whether its
// frequency is two.
func TestStatisticsSimple2(t *testing.T) {
	stats := NewStatistics[common.Address]()
	address := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	stats.Count(address)
	stats.Count(address)
	frequency := stats.Frequency(address)
	if frequency != 2 {
		t.Fatalf("Counting failed")
	}
}

// TestStatisticsSimple3 counts the single occurrence of two addresses and checks whether
// their frequencies are one and whether they exist.
func TestStatisticsSimple3(t *testing.T) {
	stats := NewStatistics[common.Address]()
	address1 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F272")
	address2 := common.HexToAddress("0xdEcAf0562A19C9fFf21c9cEB476B2858E6f1F273")
	stats.Count(address1)
	stats.Count(address2)
	frequency1 := stats.Frequency(address1)
	frequency2 := stats.Frequency(address2)
	if frequency1 != 1 || frequency2 != 1 || !stats.Exists(address1) || !stats.Exists(address2) {
		t.Fatalf("Counting failed failed")
	}
}

// TestStatisticsSimple4 tests JSON output of distribution.
func TestStatisticsSimple4(t *testing.T) {
	stats := NewStatistics[int]()
	for i := 1; i <= 10; i++ {
		stats.Count(i)
	}
	stats.Count(1)
	stats.Count(10)

	// produce distribution in JSON format
	jOut, err := json.Marshal(stats.ProduceDistribution())
	if err != nil {
		t.Fatalf("Marshalling failed to produce distribution")
	}
	expected := `{"NumData":10,"TotalFreq":12,"X":[0,0.05,0.15,0.25,0.35,0.45,0.55,0.65,0.75,0.85,0.95,1],"P":[0,0.08333333333333333,0.16666666666666666,0.25,0.3333333333333333,0.41666666666666663,0.5,0.5833333333333333,0.6666666666666666,0.8333333333333333,1,1]}`
	if string(jOut) != expected {
		t.Fatalf("produced wrong JSON output")
	}

}
