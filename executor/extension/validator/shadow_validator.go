package validator

import (
	"fmt"
	"github.com/Fantom-foundation/Aida/state/proxy"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeShadowDbValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if !cfg.Validate || !cfg.ShadowDb {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return makeShadowDbValidator(cfg, logger.NewLogger(cfg.LogLevel, "ShadowDbValidator"))
}

func makeShadowDbValidator(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &shadowDbValidator{
		cfg: cfg,
		log: log,
	}
}

type shadowDbValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg             *utils.Config
	log             logger.Logger
	overall, passed int
}

func (e *shadowDbValidator) PostTransaction(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	// cast ctx.State to ShadowStateDb
	dbWrapper, ok := ctx.State.(*proxy.ShadowStateDb)
	if !ok {
		return fmt.Errorf("internal error: state is not a shadow state db")
	}

	want := dbWrapper.GetShadowDB().GetHash()
	got := ctx.State.GetHash()

	// cast state.Data to stJSON
	c := s.Data.(*ethtest.StJSON)

	if got != want {
		err := fmt.Errorf("%v - (%v) SHADOWDB FAIL\ndifferent hashes\ngot: %v\nwant:%v", c.TestLabel, c.UsedNetwork, got.Hex(), want.Hex())
		if e.cfg.ContinueOnFailure {
			e.log.Error(err)
		} else {
			return err
		}
	} else {
		e.passed++
		e.log.Noticef("%v - (%v) SHADOWDB PASS\nblock: %v; tx: %v\nhash:%v", c.TestLabel, c.UsedNetwork, s.Block, s.Transaction, got.Hex())
	}

	e.overall++
	return nil
}

func (e *shadowDbValidator) PostRun(executor.State[txcontext.TxContext], *executor.Context, error) error {
	e.log.Noticef("SHADOWDB %v/%v tests passed.", e.passed, e.overall)
	return nil
}
