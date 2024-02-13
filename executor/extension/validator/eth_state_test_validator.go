package validator

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/ethtest/state_test"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeEthStateTestValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if !cfg.Validate {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return makeEthStateTestValidator(cfg, logger.NewLogger(cfg.LogLevel, "EthStateTestValidator"))
}

func makeEthStateTestValidator(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &ethStateTestValidator{
		cfg: cfg,
		log: log,
	}
}

type ethStateTestValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg             *utils.Config
	log             logger.Logger
	overall, passed int
}

func (e *ethStateTestValidator) PreTransaction(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	err := validateWorldState(e.cfg, ctx.State, s.Data.GetInputState(), e.log)
	if err != nil {
		return fmt.Errorf("pre alloc validation failed; %v", err)
	}

	return nil
}

func (e *ethStateTestValidator) PostTransaction(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	want := s.Data.GetStateHash()
	got := ctx.State.GetHash()

	// cast state.Data to stJSON
	c := s.Data.(*state_test.StJSON)

	if got != want {
		err := fmt.Errorf("%v - (%v) FAIL\ndifferent hashes\ngot: %v\nwant:%v", c.TestLabel, c.UsedNetwork, got.Hex(), want.Hex())
		if e.cfg.ContinueOnFailure {
			e.log.Error(err)
		} else {
			return err
		}
	} else {
		e.passed++
		e.log.Noticef("%v - (%v) PASS\nblock: %v; tx: %v\nhash:%v", c.TestLabel, c.UsedNetwork, s.Block, s.Transaction, got.Hex())
	}

	e.overall++
	return nil
}

func (e *ethStateTestValidator) PostRun(executor.State[txcontext.TxContext], *executor.Context, error) error {
	e.log.Noticef("%v/%v tests passed.", e.passed, e.overall)
	return nil
}
