package ethtest

import (
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"golang.org/x/exp/maps"
)

var usableForks = map[string]struct{}{
	"Cancun":       {},
	"Shanghai":     {},
	"Paris":        {},
	"Bellatrix":    {},
	"GrayGlacier":  {},
	"ArrowGlacier": {},
	"Altair":       {},
	"London":       {},
	"Berlin":       {},
	"Istanbul":     {},
	"MuirGlacier":  {},
	"TestNetwork":  {},
}

// NewTestCaseSplitter opens all JSON tests within path
func NewTestCaseSplitter(cfg *utils.Config) (*TestCaseSplitter, error) {
	tests, err := getTestsWithinPath(cfg, utils.StateTests)
	if err != nil {
		return nil, err
	}
	log := logger.NewLogger(cfg.LogLevel, "eth-test-decoder")

	return &TestCaseSplitter{
		enabledForks: sortForks(log, cfg.Forks),
		log:          log,
		jsons:        tests,
	}, nil
}

func sortForks(log logger.Logger, cfgForks []string) (forks []string) {
	if len(cfgForks) == 1 && strings.ToLower(cfgForks[0]) == "all" {
		forks = maps.Keys(usableForks)
	} else {
		for _, fork := range cfgForks {
			fork = strings.Replace(strings.ToLower(fork), "glacier", "Glacier", -1)
			fork = strings.Title(fork)
			if _, ok := usableForks[fork]; !ok {
				log.Warningf("Unknown name fork name %v, removing", fork)
				continue
			}
			forks = append(forks, fork)
		}
	}
	return forks
}

type TestCaseSplitter struct {
	enabledForks []string  // Which forks are enabled by user (default is all)
	jsons        []*stJSON // Decoded json files
	log          logger.Logger
}

// SplitStateTests iterates unmarshalled Geth-State test-files and divides them by 1) fork and
// 2) tests cases. Each file contains 1..N enabledForks, one block environment (marked as 'env') and one
// input alloc (marked as 'env'). Each fork within a file contains 1..N tests (marked as 'post').
func (s *TestCaseSplitter) SplitStateTests() (dividedTests []txcontext.TxContext) {
	var overall uint32

	// Iterate all JSONs
	for _, stJson := range s.jsons {
		number := 0
		env := &stJson.Env
		baseFee := stJson.Env.BaseFee
		if baseFee == nil {
			// ethereum uses `0x10` for genesis baseFee. Therefore, it defaults to
			// parent - 2 : 0xa as the basefee for 'this' context.
			baseFee = &BigInt{*big.NewInt(0x0a)}
		}

		// TODO: each test requires its own block context and chain config
		// Iterate all usable forks within one JSON file
		for _, fork := range s.enabledForks {
			posts := stJson.Post[fork]
			// Iterate all tests within one fork
			for _, post := range posts {
				number++
				msg, err := stJson.Tx.toMessage(post, baseFee)
				if err != nil {
					s.log.Warningf("Path: %v, fork: %v, test number: %v\n"+
						"cannot decode tx to message: %v", stJson.path, fork, number, err)
					continue
				}

				dividedTests = append(dividedTests, newStateTestTxContest(stJson, env, msg, post.RootHash, fork, number))
				overall++
			}
		}
	}

	s.log.Noticef("Found %v runnable state tests...", overall)

	return dividedTests
}
