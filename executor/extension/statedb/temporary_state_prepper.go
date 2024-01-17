package statedb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
	statedb "github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeTemporaryStatePrepper creates an executor.Extension which Makes a fresh StateDb
// after each transaction. Default is offTheChainStateDb.
// NOTE: inMemoryStateDb currently does not work for block 67m onwards.
func MakeTemporaryStatePrepper(cfg *utils.Config) executor.Extension[substate_transaction.SubstateData] {
	switch cfg.DbImpl {
	case "in-memory", "memory":
		return temporaryInMemoryStatePrepper{}
	case "off-the-chain":
		fallthrough
	default:
		// offTheChainStateDb is default value
		substate.RecordReplay = true
		return &temporaryOffTheChainStatePrepper{
			cfg: cfg,
		}
	}
}

// temporaryInMemoryStatePrepper is an extension that introduces a fresh in-memory
// StateDB instance before each transaction execution.
type temporaryInMemoryStatePrepper struct {
	extension.NilExtension[substate_transaction.SubstateData]
}

// PreTransaction creates new fresh StateDb
func (temporaryInMemoryStatePrepper) PreTransaction(state executor.State[substate_transaction.SubstateData], ctx *executor.Context) error {
	alloc := state.Data.GetInputAlloc()
	ctx.State = statedb.MakeInMemoryStateDB(alloc, uint64(state.Block))
	return nil
}

// temporaryOffTheChainStatePrepper is an extension that introduces a fresh offTheChain
// StateDB instance before each transaction execution.
type temporaryOffTheChainStatePrepper struct {
	extension.NilExtension[substate_transaction.SubstateData]
	cfg *utils.Config
}

// PreTransaction creates new fresh StateDb
func (p *temporaryOffTheChainStatePrepper) PreTransaction(state executor.State[substate_transaction.SubstateData], ctx *executor.Context) error {
	var err error
	if p.cfg == nil {
		return fmt.Errorf("temporaryOffTheChainStatePrepper: cfg is nil")
	}
	ctx.State, err = statedb.MakeOffTheChainStateDB(state.Data.GetInputAlloc(), uint64(state.Block), statedb.NewChainConduit(p.cfg.ChainID == utils.EthereumChainID, utils.GetChainConfig(utils.EthereumChainID)))
	return err
}
