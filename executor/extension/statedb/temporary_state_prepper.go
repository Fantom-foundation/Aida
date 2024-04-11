package statedb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	statedb "github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/state"
)

// MakeTemporaryStatePrepper creates an executor.Extension which Makes a fresh StateDb
// after each txcontext. Default is offTheChainStateDb.
// NOTE: inMemoryStateDb currently does not work for block 67m onwards.
func MakeTemporaryStatePrepper(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	switch cfg.DbImpl {
	case "in-memory", "memory":
		return temporaryInMemoryStatePrepper{}
	case "off-the-chain":
		fallthrough
	default:
		// offTheChainStateDb is default value
		state.EnableRecordReplay()
		return &temporaryOffTheChainStatePrepper{
			cfg: cfg,
		}
	}
}

// temporaryInMemoryStatePrepper is an extension that introduces a fresh in-memory
// StateDB instance before each transaction execution.
type temporaryInMemoryStatePrepper struct {
	extension.NilExtension[txcontext.TxContext]
}

func (temporaryInMemoryStatePrepper) PreTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	alloc := state.Data.GetInputState()
	ctx.State = statedb.MakeInMemoryStateDB(alloc, uint64(state.Block))
	return nil
}

// temporaryOffTheChainStatePrepper is an extension that introduces a fresh offTheChain
// StateDB instance before each transaction execution.
type temporaryOffTheChainStatePrepper struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
}

func (p *temporaryOffTheChainStatePrepper) PreTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	var err error
	if p.cfg == nil {
		return fmt.Errorf("temporaryOffTheChainStatePrepper: cfg is nil")
	}
	ctx.State, err = statedb.MakeOffTheChainStateDB(state.Data.GetInputState(), uint64(state.Block), statedb.NewChainConduit(p.cfg.ChainID == utils.EthereumChainID, utils.GetChainConfig(utils.EthereumChainID)))
	return err
}
