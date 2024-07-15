package ethtest

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

// NewDecoder opens all JSON tests within path
func NewDecoder(cfg *utils.Config) (*Decoder, error) {
	tests, err := getTestsWithinPath(cfg, utils.StateTests)
	if err != nil {
		return nil, err
	}

	return &Decoder{
		cfg:   cfg,
		log:   logger.NewLogger(cfg.LogLevel, "eth-test-decoder"),
		jsons: tests,
	}, nil
}

type Decoder struct {
	cfg   *utils.Config
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
		for fork, posts := range stJson.Post {
			if _, ok := usableForks[fork]; !ok {
				continue
			}
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
