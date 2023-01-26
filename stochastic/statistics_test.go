package stochastic

import (
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
