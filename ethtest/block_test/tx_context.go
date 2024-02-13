package blocktest

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type txContext struct {
	env  *BlockEnvironment
	msg  types.Message
	post core.GenesisAlloc
	Cfg  *Config
	pre  core.GenesisAlloc
}

func (d *txContext) GetStateHash() common.Hash {
	// ignored
	return common.Hash{}
}

func (d *txContext) GetInputState() txcontext.WorldState {
	return NewGethWorldState(d.pre)
}

func (d *txContext) GetBlockEnvironment() txcontext.BlockEnvironment {
	return d.env
}

func (d *txContext) GetMessage() core.Message {
	return d.msg
}

func (d *txContext) GetOutputState() txcontext.WorldState {
	return NewGethWorldState(d.post)
}

func (d *txContext) GetReceipt() txcontext.Receipt {
	//TODO implement me
	panic("implement me")
}

func NewData(block BtBlock, tx *Transaction, bt *BtJSON) txcontext.TxContext {
	return &txContext{
		env:  block.BlockHeader,
		msg:  tx.ToMessage(),
		pre:  bt.Pre,
		post: bt.Post,
		Cfg: &Config{
			Network:    bt.Network,
			SealEngine: bt.SealEngine,
		},
	}
}

type Config struct {
	Network    string
	SealEngine string
}
