package txgenerator

import (
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewNormaTxContext creates a new transaction context for a norma transaction.
// It expects a signed transaction.
func NewNormaTxContext(tx *types.Transaction, blkNumber uint64) (txcontext.TxContext, error) {
	// extract sender from tx by passing it through the signer
	sender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return nil, err
	}
	return &normaTxData{
		txData: txData{
			Env: normaTxBlockEnv{
				blkNumber: blkNumber,
			},
			Message: types.NewMessage(
				sender,
				tx.To(),
				tx.Nonce(),
				tx.Value(),
				tx.Gas(),
				tx.GasPrice(),
				tx.GasFeeCap(),
				tx.GasTipCap(),
				tx.Data(),
				tx.AccessList(),
				false,
			),
		},
	}, nil
}

// normaTxData is a transaction context for norma transactions.
type normaTxData struct {
	txData
}

// normaTxBlockEnv is a block environment for norma transactions.
type normaTxBlockEnv struct {
	blkNumber uint64
}

// GetCoinbase returns the coinbase address.
func (e normaTxBlockEnv) GetCoinbase() common.Address {
	return common.HexToAddress("0x1")
}

// GetDifficulty returns the current difficulty level.
func (e normaTxBlockEnv) GetDifficulty() *big.Int {
	return big.NewInt(1)
}

// GetGasLimit returns the maximum amount of gas that can be used in a block.
func (e normaTxBlockEnv) GetGasLimit() uint64 {
	return 1_000_000_000_000
}

// GetNumber returns the current block number.
func (e normaTxBlockEnv) GetNumber() uint64 {
	return e.blkNumber
}

// GetTimestamp returns the timestamp of the current block.
func (e normaTxBlockEnv) GetTimestamp() uint64 {
	// use current timestamp as the block timestamp
	// since we don't have a real block
	return uint64(time.Now().Unix())
}

// GetBlockHash returns the hash of the block with the given number.
func (e normaTxBlockEnv) GetBlockHash(blockNumber uint64) (common.Hash, error) {
	// transform the block number into a hash
	// we don't have real block hashes, so we just use the block number
	return common.BigToHash(big.NewInt(int64(blockNumber))), nil
}

// GetBaseFee returns the base fee for transactions in the current block.
func (e normaTxBlockEnv) GetBaseFee() *big.Int {
	return big.NewInt(0)
}
