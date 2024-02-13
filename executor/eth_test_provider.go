package executor

import (
	"fmt"

	statetest "github.com/Fantom-foundation/Aida/ethtest"
	blocktest "github.com/Fantom-foundation/Aida/ethtest/block_test"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func NewEthStateTestProvider(cfg *utils.Config) Provider[txcontext.TxContext] {
	return ethTestProvider{cfg}
}

type ethTestProvider struct {
	cfg *utils.Config
}

func (e ethTestProvider) Run(_ int, _ int, consumer Consumer[txcontext.TxContext]) error {
	b, err := statetest.OpenBlockTests(e.cfg.ArgPath)
	if err != nil {
		return err
	}

	// iterate all tests
	for _, bt := range b {
		// iterate blocks inside tests
		for blockCounter, blk := range bt.Blocks {
			// iterate txs inside block
			for txCounter, tx := range blk.Transactions {
				err = consumer(TransactionInfo[txcontext.TxContext]{
					Block:       blockCounter + 1, // zero block is genesis
					Transaction: txCounter,
					Data:        blocktest.NewData(blk, tx, bt),
				})
			}
			fmt.Println(blockCounter)
		}

	}

	// iterate all state json files
	//for _, t := range b {
	//	// divide them by fork
	//	for i, dt := range t.Divide(e.cfg.ChainID) {
	//		err = consumer(TransactionInfo[txcontext.TxContext]{
	//			Block:       int(dt.env.GetNumber()),
	//			Transaction: i,
	//			Data:        dt,
	//		})
	//	}
	//}

	return nil
}

func (e ethTestProvider) Close() {
	// ignored
}
