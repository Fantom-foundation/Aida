package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

func NewChainConduit(isEthereum bool, chainConfig *params.ChainConfig) *ChainConduit {
	return &ChainConduit{
		isEthereum:  isEthereum,
		chainConfig: chainConfig,
	}
}

// ChainConduit is used to determine special behaviour between Opera and Ethereum (and their hard forks different behaviours) (e.g. EndTransaction).
type ChainConduit struct {
	isEthereum  bool
	chainConfig *params.ChainConfig
}

func (c *ChainConduit) IsFinalise(block *big.Int) bool {
	if !c.isEthereum {
		return true
	} else {
		return c.chainConfig.IsByzantium(block)
	}
}

func (c *ChainConduit) DeleteEmptyObjects(block *big.Int) bool {
	if !c.isEthereum {
		return true
	} else {
		return c.chainConfig.IsByzantium(block) || c.chainConfig.IsEIP158(block)
	}
}
