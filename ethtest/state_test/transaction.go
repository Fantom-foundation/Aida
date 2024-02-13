package state_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/Aida/ethtest/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type stTransaction struct {
	GasPrice             *util.BigInt        `json:"gasPrice"`
	MaxFeePerGas         *util.BigInt        `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *util.BigInt        `json:"maxPriorityFeePerGas"`
	Nonce                *util.BigInt        `json:"nonce"`
	To                   string              `json:"to"`
	Data                 []string            `json:"data"`
	AccessLists          []*types.AccessList `json:"accessLists,omitempty"`
	GasLimit             []*util.BigInt      `json:"gasLimit"`
	Value                []string            `json:"value"`
	PrivateKey           hexutil.Bytes       `json:"secretKey"`
}

func (tx *stTransaction) toMessage(ps stPostState, baseFee *util.BigInt) (*types.Message, error) {
	// Derive sender from private key if present.
	var from common.Address
	if len(tx.PrivateKey) > 0 {
		key, err := crypto.ToECDSA(tx.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %v", err)
		}
		from = crypto.PubkeyToAddress(key.PublicKey)
	}
	// Parse recipient if present.
	var to *common.Address
	if tx.To != "" {
		to = new(common.Address)
		if err := to.UnmarshalText([]byte(tx.To)); err != nil {
			return nil, fmt.Errorf("invalid to address: %v", err)
		}
	}

	// Get values specific to this post state.
	if ps.indexes.Data > len(tx.Data) {
		return nil, fmt.Errorf("tx data index %d out of bounds", ps.indexes.Data)
	}
	if ps.indexes.Value > len(tx.Value) {
		return nil, fmt.Errorf("tx value index %d out of bounds", ps.indexes.Value)
	}
	if ps.indexes.Gas > len(tx.GasLimit) {
		return nil, fmt.Errorf("tx gas limit index %d out of bounds", ps.indexes.Gas)
	}
	dataHex := tx.Data[ps.indexes.Data]
	valueHex := tx.Value[ps.indexes.Value]
	gasLimit := tx.GasLimit[ps.indexes.Gas]
	// Value, Data hex encoding is messy: https://github.com/ethereum/tests/issues/203
	value := new(big.Int)
	if valueHex != "0x" {
		v, ok := math.ParseBig256(valueHex)
		if !ok {
			return nil, fmt.Errorf("invalid tx value %q", valueHex)
		}
		value = v
	}
	data, err := hex.DecodeString(strings.TrimPrefix(dataHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid tx data %q", dataHex)
	}
	var accessList types.AccessList
	if tx.AccessLists != nil && tx.AccessLists[ps.indexes.Data] != nil {
		accessList = *tx.AccessLists[ps.indexes.Data]
	}
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	gasPrice := tx.GasPrice
	if baseFee != nil {
		if tx.MaxFeePerGas == nil {
			tx.MaxFeePerGas = gasPrice
		}
		if tx.MaxFeePerGas == nil {
			tx.MaxFeePerGas = new(util.BigInt)
		}
		if tx.MaxPriorityFeePerGas == nil {
			tx.MaxPriorityFeePerGas = tx.MaxFeePerGas
		}
		gasPrice = &util.BigInt{*math.BigMin(new(big.Int).Add(tx.MaxPriorityFeePerGas.Convert(), baseFee.Convert()),
			tx.MaxFeePerGas.Convert())}
	}
	if gasPrice == nil {
		return nil, fmt.Errorf("no gas price provided")
	}

	msg := types.NewMessage(from, to, tx.Nonce.Uint64(), value, gasLimit.Uint64(), gasPrice.Convert(),
		tx.MaxFeePerGas.Convert(), tx.MaxPriorityFeePerGas.Convert(), data, accessList, false)
	return &msg, nil
}
