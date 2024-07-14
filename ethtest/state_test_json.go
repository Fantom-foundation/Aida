package ethtest

import (
	"math/big"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

var usableForks = map[string]struct{}{
	"Cancun":        {},
	"Shanghai":      {},
	"Paris":         {},
	"Bellatrix":     {},
	"Gray Glacier":  {},
	"Arrow Glacier": {},
	"Altair":        {},
	"London":        {},
	"Berlin":        {},
	"Istanbul":      {},
	"MuirGlacier":   {},
	"TestNetwork":   {},
}

// stJSON serves as a 'middleman' into which are data unmarshalled from geth test files.
type stJSON struct {
	path string
	Env  stBlockEnvironment  `json:"env"`
	Pre  types.GenesisAlloc  `json:"pre"`
	Tx   stTransaction       `json:"transaction"`
	Out  hexutil.Bytes       `json:"out"`
	Post map[string][]stPost `json:"post"`
}

func (s *stJSON) setPath(path string) {
	s.path = path
}

// stPost indicates data for each transaction.
type stPost struct {
	// RootHash holds expected state hash after a transaction is executed.
	RootHash common.Hash `json:"hash"`
	// LogsHash holds expected logs hash (Bloom) after a transaction is executed.
	LogsHash        common.Hash   `json:"logs"`
	TxBytes         hexutil.Bytes `json:"txbytes"`
	ExpectException string        `json:"expectException"`
	indexes         Index
}

// Index indicates position of data, gas, value for executed transaction.
type Index struct {
	Data  int `json:"data"`
	Gas   int `json:"gas"`
	Value int `json:"value"`
}

// DivideStateTests iterates unmarshalled Geth-State test-files and divides them by 1) fork and
// 2) tests cases. Each file contains 1..N forks, one block environment (marked as 'env') and one
// input alloc (marked as 'env'). Each fork within a file contains 1..N tests (marked as 'post').
func (d *GethTestDecoder) DivideStateTests() (dividedTests []txcontext.TxContext) {
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
