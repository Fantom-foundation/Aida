package state

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	carmen "github.com/Fantom-foundation/Carmen/go/state"
)

type CarmenStateTestCase struct {
	Variant string
	Schema  int
	Archive string
}

func (c CarmenStateTestCase) String() string {
	return fmt.Sprintf("DB Variant: %s, Schema: %d, Archive type: %v", c.Variant, c.Schema, c.Archive)
}

// All combinations of carmen db configuration for testing db creation/close function in Aida
func GetAllCarmenConfigurations() []CarmenStateTestCase {
	archives := []string{
		"none",
		"leveldb",
		"sqlite",
		"s4",
		"s5",
	}

	var testCases []CarmenStateTestCase

	for _, variant := range carmen.GetAllVariants() {
		for _, schema := range carmen.GetAllSchemas() {
			for _, archive := range archives {
				testCases = append(testCases, CarmenStateTestCase{
					Variant: string(variant),
					Schema:  int(schema),
					Archive: archive,
				})
			}
		}
	}

	return testCases
}

// A combination of carmen db configuration for testing interface
func GetCarmenStateTestCases() []CarmenStateTestCase {
	archives := []string{
		"none",
		"leveldb",
	}

	variant := "go-file"
	schema := 3

	var testCases []CarmenStateTestCase

	for _, archive := range archives {
		testCases = append(testCases, CarmenStateTestCase{
			Variant: variant,
			Schema:  schema,
			Archive: archive,
		})
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
