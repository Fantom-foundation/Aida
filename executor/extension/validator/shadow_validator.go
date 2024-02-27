package validator

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeShadowDbValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if !cfg.ShadowDb {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return makeShadowDbValidator(cfg)
}

func makeShadowDbValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return &shadowDbValidator{
		cfg: cfg,
	}
}

type shadowDbValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg *utils.Config
}

func (e *shadowDbValidator) PostBlock(_ executor.State[txcontext.TxContext], ctx *executor.Context) error {
	// Retrieve hash from the state, if this there is mismatch between prime and shadow db error is returned
	ctx.State.GetHash()
	return ctx.State.Error()
}
