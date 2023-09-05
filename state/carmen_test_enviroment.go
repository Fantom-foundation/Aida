package state

import (
	"math/rand"
	"testing"
	"time"

	carmen "github.com/Fantom-foundation/Carmen/go/state"
)

type CarmenStateTestCase struct {
	Variant string
	Archive string
}

func GetCarmenStateTestCases() []CarmenStateTestCase {
	variants := []string{""}
	for _, variant := range carmen.GetAllVariants() {
		variants = append(variants, string(variant))
	}

	archives := []string{
		"none",
		"leveldb",
		"sqlite",
		"s4",
		"s5",
	}

	var testCases []CarmenStateTestCase

	for _, variant := range variants {
		for _, archive := range archives {
			testCases = append(testCases, CarmenStateTestCase{Variant: variant, Archive: archive})
		}
	}

	return testCases
}

// MakeRandomByteSlice creates byte slice of given length with randomized values
func MakeRandomByteSlice(t *testing.T, bufferLength int) []byte {
	// make byte slice
	buffer := make([]byte, bufferLength)

	// fill the slice with random data
	_, err := rand.Read(buffer)
	if err != nil {
		t.Fatalf("failed test data; can not generate random byte slice; %s", err.Error())
	}

	return buffer
}

func GetRandom(rangeLower int, rangeUpper int) int {
	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	randInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)
	return randInt
}
