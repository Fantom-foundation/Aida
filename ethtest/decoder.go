package ethtest

import (
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"golang.org/x/exp/maps"
)

// NewDecoder opens all JSON tests within path
func NewDecoder(cfg *utils.Config) (*Decoder, error) {
	tests, err := getTestsWithinPath(cfg, utils.StateTests)
	if err != nil {
		return nil, err
	}
	log := logger.NewLogger(cfg.LogLevel, "eth-test-decoder")

	return &Decoder{
		forks: sortForks(log, cfg.Forks),
		log:   log,
		jsons: tests,
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

type Decoder struct {
	forks []string
	log   logger.Logger
	jsons []*stJSON
}

// DivideStateTests iterates unmarshalled Geth-State test-files and divides them by 1) fork and
// 2) tests cases. Each file contains 1..N forks, one block environment (marked as 'env') and one
// input alloc (marked as 'env'). Each fork within a file contains 1..N tests (marked as 'post').
func (d *Decoder) DivideStateTests() (dividedTests []txcontext.TxContext) {
	var overall uint32
	// Iterate all JSONs
	for _, stJson := range d.jsons {
		env := &stJson.Env
		baseFee := stJson.Env.BaseFee
		if baseFee == nil {
			// ethereum uses `0x10` for genesis baseFee. Therefore, it defaults to
			// parent - 2 : 0xa as the basefee for 'this' context.
			baseFee = &BigInt{*big.NewInt(0x0a)}
		}

		// Iterate all usable forks within one JSON file
		for _, fork := range d.forks {
			posts := stJson.Post[fork]
			// Iterate all tests within one fork
			for i, post := range posts {
				msg, err := stJson.Tx.toMessage(post, baseFee)
				if err != nil {
					d.log.Warningf("Path: %v, fork: %v, test number: %v\n"+
						"cannot decode tx to message: %v", stJson.path, fork, i, err)
					continue
				}

				dividedTests = append(dividedTests, newStateTestTxContest(stJson, env, msg, post.RootHash, fork, i))
				overall++
			}
		}
	}

	d.log.Noticef("Found %v runnable state tests...", overall)

	return dividedTests
}
