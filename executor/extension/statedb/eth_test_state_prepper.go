package statedb

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	statedb "github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

func NewTemporaryEthStatePrepper(cfg *utils.Config) executor.Extension[*ethtest.Data] {
	return &ethStatePrepper{
		cfg: cfg,
	}
}

type ethStatePrepper struct {
	extension.NilExtension[*ethtest.Data]
	cfg *utils.Config
}

func (e ethStatePrepper) PreTransaction(state executor.State[*ethtest.Data], ctx *executor.Context) error {
	db := rawdb.NewMemoryDatabase()
	_, err := state.Data.Genesis.Commit(db)
	if err != nil {
		return err
	}

	ctx.State = statedb.NewInMemoryGethStateDB(
		statedb.NewOffTheChainStateDB(db),
		statedb.NewChainConduit(e.cfg.ChainID == utils.EthereumChainID, utils.GetChainConfig(utils.EthereumChainID)),
		new(big.Int).SetUint64(uint64(state.Block)),
	)
	return nil
}

func (e ethStatePrepper) PostTransaction(state executor.State[*ethtest.Data], ctx *executor.Context) error {
	// validate post state accounts in test file against what we have in state db
	for addr, acct := range state.Data.Post {
		// address is indirectly verified by the other fields, as it's the db key
		code2 := ctx.State.GetCode(addr)
		balance2 := ctx.State.GetBalance(addr)
		nonce2 := ctx.State.GetNonce(addr)
		if !bytes.Equal(code2, acct.Code) {
			return fmt.Errorf("account code mismatch for addr: %s want: %v have: %s", addr, acct.Code, hex.EncodeToString(code2))
		}
		if balance2.Cmp(acct.Balance) != 0 {
			return fmt.Errorf("account balance mismatch for addr: %s, want: %d, have: %d", addr, acct.Balance, balance2)
		}
		if nonce2 != acct.Nonce {
			return fmt.Errorf("account nonce mismatch for addr: %s want: %d have: %d", addr, acct.Nonce, nonce2)
		}
	}
	return nil

}
