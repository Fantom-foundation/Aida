// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package txgenerator

import (
	"math/big"
	"time"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewNormaTxContext creates a new transaction context for a norma transaction.
// It expects a signed transaction if sender is nil.
func NewNormaTxContext(tx *types.Transaction, blkNumber uint64, sender *common.Address, fork string) (txcontext.TxContext, error) {
	s := common.Address{}
	if sender == nil {
		addr, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
		if err != nil {
			return nil, err
		}
		s = addr
	} else {
		s = *sender
	}
	return &normaTxData{
		txData: txData{
			Env: normaTxBlockEnv{
				blkNumber: blkNumber,
				fork:      fork,
			},
			Message: &core.Message{
				To:                tx.To(),
				From:              s,
				Nonce:             tx.Nonce(),
				Value:             tx.Value(),
				GasLimit:          tx.Gas(),
				GasPrice:          tx.GasPrice(),
				GasFeeCap:         tx.GasFeeCap(),
				GasTipCap:         tx.GasTipCap(),
				Data:              tx.Data(),
				AccessList:        tx.AccessList(),
				SkipAccountChecks: false,
				BlobGasFeeCap:     tx.BlobGasFeeCap(),
				BlobHashes:        tx.BlobHashes(),
			},
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
	fork      string
}

// GetRandom is not used in Norma Tx-Generator.
func (e normaTxBlockEnv) GetRandom() *common.Hash {
	return nil
}

// GetCoinbase returns the coinbase address.
func (e normaTxBlockEnv) GetCoinbase() common.Address {
	return common.HexToAddress("0x1")
}

// GetBlobBaseFee is not used in Norma Tx-Generator.
func (e normaTxBlockEnv) GetBlobBaseFee() *big.Int {
	return big.NewInt(0)
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

func (e normaTxBlockEnv) GetFork() string {
	return e.fork
}
