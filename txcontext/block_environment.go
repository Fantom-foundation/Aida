package txcontext

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// BlockEnvironment represents an interface for retrieving and modifying Ethereum-like blockchain environment information.
type BlockEnvironment interface {
	// GetCoinbase returns the coinbase address.
	GetCoinbase() common.Address

	// GetDifficulty returns the current difficulty level.
	GetDifficulty() *big.Int

	// GetGasLimit returns the maximum amount of gas that can be used in a block.
	GetGasLimit() uint64

	// GetNumber returns the current block number.
	GetNumber() uint64

	// GetTimestamp returns the timestamp of the current block.
	GetTimestamp() uint64

	// GetBlockHash returns the hash of the block with the given number.
	GetBlockHash(blockNumber uint64) common.Hash

	// GetBaseFee returns the base fee for transactions in the current block.
	GetBaseFee() *big.Int
}
