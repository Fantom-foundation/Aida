package executor

import (
	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/utils"
)

// blockHeader is env?
// []transactions is message

// geth/tests/block_test.go for blockchain tests
// geth/tests/block_test_util.go for unmarshalling

func NewEthTestProvider(cfg *utils.Config) Provider[*ethtest.Data] {
	return ethTestProvider{cfg}
}

type ethTestProvider struct {
	cfg *utils.Config
}

func (e ethTestProvider) Run(_ int, _ int, consumer Consumer[*ethtest.Data]) error {
	// todo redo to a dir
	b, err := ethtest.Open(e.cfg.ArgPath)
	if err != nil {
		return err
	}

	// iterate all bt json files
	for _, bt := range b {
		// iterate all blocks inside one file
		for blockNumber, block := range bt.Blocks {
			// iterate all transactions inside one block
			for txNumber, tx := range block.Transactions {
				err = consumer(TransactionInfo[*ethtest.Data]{
					Block:       blockNumber,
					Transaction: txNumber,
					Data:        ethtest.NewData(block, tx, bt, e.cfg.ChainID),
				})
				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func (e ethTestProvider) Close() {
	//TODO implement me
	panic("implement me")
}
