package statedb

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeEthStateTestDbPrepper(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeEthStateTestDbPrepper(logger.NewLogger(cfg.LogLevel, "EthStatePrepper"), cfg)
}

func makeEthStateTestDbPrepper(log logger.Logger, cfg *utils.Config) *ethStateTestDbPrepper {
	return &ethStateTestDbPrepper{
		cfg: cfg,
		log: log,
	}
}

type ethStateTestDbPrepper struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
	log logger.Logger
}

func (e ethStateTestDbPrepper) PreTransaction(st executor.State[txcontext.TxContext], ctx *executor.Context) error {
	var err error
	cfg := e.cfg
	// We reduce the node cache size to be used by Carmen to 1 MB
	// reduce the cache creation and flush time, and thus to improve
	// the processing time of each transaction.
	cfg.CarmenStateCacheSize = 1000
	cfg.CarmenNodeCacheSize = (16 << 20) // = 16 MiB
	ctx.State, ctx.StateDbPath, err = utils.PrepareStateDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to prepare statedb; %v", err)
	}

	return nil
}

func (e ethStateTestDbPrepper) PostTransaction(_ executor.State[txcontext.TxContext], ctx *executor.Context) error {
	if ctx.State != nil {
		err := ctx.State.Close()
		if err != nil {
			return fmt.Errorf("cannot close db %v; %v", ctx.StateDbPath, err)
		}
	}

	err := os.RemoveAll(ctx.StateDbPath)
	if err != nil {
		return fmt.Errorf("cannot remove db %v; %v", ctx.StateDbPath, err)
	}

	return nil
}
