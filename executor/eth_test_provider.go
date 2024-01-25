package executor

import (
	statetest "github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

// blockHeader is env?
// []transactions is message

// geth/tests/block_test.go for blockchain tests
// geth/tests/block_test_util.go for unmarshalling

func NewEthStateTestProvider(cfg *utils.Config) Provider[txcontext.TxContext] {
	return ethTestProvider{cfg}
}

type ethTestProvider struct {
	cfg *utils.Config
}

func (e ethTestProvider) Run(_ int, _ int, consumer Consumer[txcontext.TxContext]) error {
	// todo redo to a dir
	b, err := statetest.OpenStateTests(e.cfg.ArgPath)
	if err != nil {
		return err
	}

	// iterate all state json files
	for _, t := range b {
		// divide them by fork
		for i, dt := range t.Divide(e.cfg.ChainID) {
			err = consumer(TransactionInfo[txcontext.TxContext]{
				Block:       int(dt.Env.GetNumber()),
				Transaction: i,
				Data:        dt,
			})
		}
	}

	return nil
}

func (e ethTestProvider) Close() {
	// ignored
}
