package executor

import (
	"math"

	statetest "github.com/Fantom-foundation/Aida/ethtest/state_test"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

// blockHeader is env?
// []transactions is message

// geth/tests/block_test.go for blockchain tests
// geth/tests/block_test_util.go for unmarshalling

func NewEthTestProvider(cfg *utils.Config) Provider[txcontext.TxContext] {
	return ethTestProvider{cfg}
}

type ethTestProvider struct {
	cfg *utils.Config
}

func (e ethTestProvider) Run(_ int, _ int, consumer Consumer[txcontext.TxContext]) error {
	// todo redo to a dir
	b, err := statetest.Open(e.cfg.ArgPath)
	if err != nil {
		return err
	}

	// iterate all bt json files
	for _, bt := range b {
		//fmt.Println(bt)
		err = consumer(TransactionInfo[txcontext.TxContext]{
			Block:       math.MaxInt,
			Transaction: 0,
			Data:        bt,
		})
		// iterate all blocks inside one file
		//for blockNumber, block := range bt.Blocks {
		//	// iterate all transactions inside one block
		//	for txNumber, tx := range block.Transactions {
		//		err = consumer(TransactionInfo[txcontext.TxContext]{
		//			Block:       blockNumber,
		//			Transaction: txNumber,
		//			Data:        ethtest.NewData(block, tx, bt, e.cfg.ChainID),
		//		})
		//		if err != nil {
		//			return err
		//		}
		//	}
		//
		//}
	}

	return nil
}

func (e ethTestProvider) Close() {
	//TODO implement me
	panic("implement me")
}
