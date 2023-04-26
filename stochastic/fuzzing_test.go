package stochastic

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/rand"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
)

// fuzzSource is a random generator source from a fuzzing source.
type fuzzSource struct {
	buf *bytes.Reader // read buffer for fuzz string
}

// newFuzzSource creates a new fuzzing source for random number generation.
func newFuzzSource(str []byte) *fuzzSource {
	return &fuzzSource{buf: bytes.NewReader(str)}
}

// Int63() retrieves next random number from the fuzzing string.
// If the fuzzing string is depleted, Int63() returns zero.
func (s *fuzzSource) Int63() int64 {
	var result int64
	if s.buf.Len() >= 8 {
		if err := binary.Read(s.buf, binary.LittleEndian, &result); err != nil {
			panic("Reading from fuzzing string failed.")
		}
		if result < 0 {
			result = -result
		}
	}
	return result
}

// Seed is not used for the fuzzing string
func (s *fuzzSource) Seed(_ int64) {
}

// End returns true iff the end of fuzz string is reached.
func (s *fuzzSource) End() bool {
	return s.buf.Len() < 8
}

// FuzzStochastic produces a seed corpus of random strings of various sizes
func FuzzStochastic(f *testing.F) {

	// create corpus
	testcases := []int{8 * 512, 8 * 1024}
	rand.Seed(1)
	for _, n := range testcases {
		randomStr := make([]byte, n)
		if _, err := rand.Read(randomStr); err != nil {
			log.Fatalf("error producing a random byte slice. Error: %v", err)
		}
		f.Add(randomStr)
	}

	f.Fuzz(func(f *testing.T, fuzzingStr []byte) {

		// generate configuration
		cfg := utils.Config{
			ContractNumber:     1000,
			KeysNumber:         1000,
			ValuesNumber:       1000,
			SnapshotDepth:      100,
			BlockLength:        3,
			SyncPeriodLength:   10,
			OperationFrequency: 2,

			ShadowImpl:     "geth",
			StateDbTempDir: "/tmp/",
			DbImpl:         "carmen",
			DbVariant:      "go-file",
		}

		// create a directory for the store to place all its files, and
		// instantiate the state DB under testing.
		db, stateDirectory, _, err := utils.PrepareStateDB(&cfg)
		if err != nil {
			f.Errorf("failed opening StateDB. Error: %v", err)
		}
		defer os.RemoveAll(stateDirectory)

		// generate uniform events
		events := GenerateUniformRegistry(&cfg).NewEventRegistryJSON()

		// generate uniform matrix
		e := NewEstimationModelJSON(&events)

		// construct random generator from fuzzing string
		fSrc := newFuzzSource(fuzzingStr)
		rg := rand.New(fSrc)

		// create a stochastic state
		ss := createState(&cfg, &e, db, rg, utils.NewLogger("INFO", "Fuzzing Stochastic"))

		// get stochastic matrix
		operations, A, state := getStochasticMatrix(&e)

		// generate operations/random parameters from fuzzing string
		for !fSrc.End() {

			// decode opcode
			op, addrCl, keyCl, valueCl := DecodeOpcode(operations[state])

			// execute operation with its argument classes
			ss.execute(op, addrCl, keyCl, valueCl)

			// check for errors
			if err := ss.db.Error(); err != nil {
				f.Errorf("failed fuzzing. Error: %v", err)
			}

			// transit to next state in Markovian process
			state = nextState(rg, A, state)
		}
	})
}
