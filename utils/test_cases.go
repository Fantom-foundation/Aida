package utils

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

const testAccountStorageSize = 10

type StateDbTestCase struct {
	Variant        string
	ShadowImpl     string
	archiveMode    bool
	ArchiveVariant string
	primeRandom    bool
}

func GetStateDbTestCases() []StateDbTestCase {
	testCases := []StateDbTestCase{
		{"geth", "", true, "", false},
		{"geth", "geth", true, "", false},
		{"carmen", "geth", false, "none", false},
		{"carmen", "geth", true, "ldb", false},
		{"carmen", "geth", true, "sqlite", false},
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

// GetRandom generates random number in from given range
func GetRandom(rangeLower int, rangeUpper int) int {
	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	randInt := rangeLower + rand.Intn(rangeUpper-rangeLower+1)
	return randInt
}

// makeAccountStorage generates randomized account storage with testAccountStorageSize length
func makeAccountStorage(t *testing.T) map[common.Hash]common.Hash {
	// create storage map
	storage := map[common.Hash]common.Hash{}

	// fill the storage map
	for j := 0; j < testAccountStorageSize; j++ {
		k := common.BytesToHash(MakeRandomByteSlice(t, 32))
		storage[k] = common.BytesToHash(MakeRandomByteSlice(t, 32))
	}

	return storage
}

// MakeTestConfig creates a config struct for testing
func MakeTestConfig(testCase StateDbTestCase) *Config {
	cfg := &Config{
		DbLogging:      "",
		DbImpl:         testCase.Variant,
		DbVariant:      "",
		ShadowImpl:     testCase.ShadowImpl,
		ShadowVariant:  "",
		ArchiveVariant: testCase.ArchiveVariant,
		ArchiveMode:    testCase.archiveMode,
		PrimeRandom:    testCase.primeRandom,
	}

	if testCase.Variant == "flat" {
		cfg.DbVariant = "go-memory"
	}

	if testCase.primeRandom {
		cfg.PrimeThreshold = 0
		cfg.RandomSeed = int64(GetRandom(1_000_000, 100_000_000))
	}

	return cfg
}

// MakeWorldState generates randomized world state containing 100 accounts
func MakeWorldState(t *testing.T) (substate.SubstateAlloc, []common.Address) {
	// create list of addresses
	var addrList []common.Address

	// create world state
	ws := substate.SubstateAlloc{}

	for i := 0; i < 100; i++ {
		// create random address
		addr := common.BytesToAddress(MakeRandomByteSlice(t, 40))

		// add to address list
		addrList = append(addrList, addr)

		// create account
		ws[addr] = &substate.SubstateAccount{
			Nonce:   uint64(GetRandom(1, 1000*5000)),
			Balance: big.NewInt(int64(GetRandom(1, 1000*5000))),
			Storage: makeAccountStorage(t),
			Code:    MakeRandomByteSlice(t, 2048),
		}
	}

	return ws, addrList
}
