package txgenerator

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func NewTxContext(env txcontext.BlockEnvironment, msg core.Message) txcontext.TxContext {
	return &txData{Env: env, Message: msg}
}

type txData struct {
	txcontext.NilTxContext
	Env     txcontext.BlockEnvironment
	Message core.Message
}

func (t *txData) GetStateHash() common.Hash {
	// ignored
	return common.Hash{}
}

func (t *txData) GetBlockEnvironment() txcontext.BlockEnvironment {
	return t.Env
}

func (t *txData) GetMessage() core.Message {
	return t.Message
}
