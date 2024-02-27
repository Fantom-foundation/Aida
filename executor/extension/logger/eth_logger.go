package logger

import (
	"github.com/Fantom-foundation/Aida/ethtest/statetest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

type ethStateTestLogger struct {
	extension.NilExtension[txcontext.TxContext]
	cfg     *utils.Config
	log     logger.Logger
	overall int
}

func MakeEthStateTestLogger(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeEthStateTestLogger(cfg, logger.NewLogger(cfg.LogLevel, "EthStateTestLogger"))
}

func makeEthStateTestLogger(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &ethStateTestLogger{
		cfg:     cfg,
		log:     log,
		overall: 0,
	}
}

// PreTransaction reports test name and fork.
func (l *ethStateTestLogger) PreTransaction(s executor.State[txcontext.TxContext], _ *executor.Context) error {
	// cast state.Data to stJSON
	c := s.Data.(*statetest.StJSON)
	l.log.Noticef("Run %v - (%v)", c.TestLabel, c.UsedNetwork)
	l.overall++
	return nil
}

// PostRun reports total tests run.
func (l *ethStateTestLogger) PostRun(executor.State[txcontext.TxContext], *executor.Context, error) error {
	l.log.Noticef("Total %v tests processed.", l.overall)
	return nil
}
