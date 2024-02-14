package validator

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeEthBlockTestValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if !cfg.Validate {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return makeEthBlockTestValidator(cfg, logger.NewLogger(cfg.LogLevel, "EthBlockTestValidator"))
}

func makeEthBlockTestValidator(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &ethBlockTestValidator{
		cfg: cfg,
		log: log,
	}
}

type ethBlockTestValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg             *utils.Config
	log             logger.Logger
	overall, passed int
}

func (e *ethBlockTestValidator) PreBlock(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	err := validateWorldState(e.cfg, ctx.State, s.Data.GetInputState(), e.log)
	if err != nil {
		return fmt.Errorf("pre alloc validation failed; %v", err)
	}

	// todo maybe check hash of block
	fmt.Println(ctx.State.GetHash())

	return nil
}

func (e *ethBlockTestValidator) PostBlock(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	err := validateWorldState(e.cfg, ctx.State, s.Data.GetOutputState(), e.log)
	if err != nil {
		return fmt.Errorf("pre alloc validation failed; %v", err)
	}

	e.overall++
	return nil
}

func (e *ethBlockTestValidator) PostRun(executor.State[txcontext.TxContext], *executor.Context, error) error {
	e.log.Noticef("%v/%v tests passed.", e.passed, e.overall)
	return nil
}