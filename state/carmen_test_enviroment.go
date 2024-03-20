package state

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Carmen/go/carmen"
	carmenstate "github.com/Fantom-foundation/Carmen/go/state"
	_ "github.com/Fantom-foundation/Carmen/go/state/cppstate"
	_ "github.com/Fantom-foundation/Carmen/go/state/gostate"
)

type CarmenStateTestCase struct {
	Variant carmen.Variant
	Schema  carmen.Schema
	Archive carmen.Archive
}

func NewCarmenStateTestCase(variant carmen.Variant, schema carmen.Schema, archive carmen.Archive) CarmenStateTestCase {
	return CarmenStateTestCase{Variant: variant, Schema: schema, Archive: archive}
}

func (c CarmenStateTestCase) String() string {
	return fmt.Sprintf("DB Variant: %s, Schema: %d, Archive type: %v", c.Variant, c.Schema, c.Archive)
}

// A combination of all carmen db configurations for testing interface
func GetAllCarmenConfigurations() []CarmenStateTestCase {
	var res []CarmenStateTestCase

	for cfg := range carmenstate.GetAllRegisteredStateFactories() {
		res = append(res, NewCarmenStateTestCase(carmen.Variant(cfg.Variant), carmen.Schema(cfg.Schema), carmen.Archive(cfg.Archive)))
	}
	return res
}

// A minimal combination of carmen db configuration for testing interface
func GetCarmenStateTestCases() []CarmenStateTestCase {
	return []CarmenStateTestCase{
		NewCarmenStateTestCase("go-file", 3, "none"),
		NewCarmenStateTestCase("go-file", 3, "leveldb"),
	}
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
