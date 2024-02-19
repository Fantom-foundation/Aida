package statetest

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
)

func CreateTestData(t *testing.T) *StJSON {
	bInt := new(big.Int).SetUint64(1)
	return &StJSON{
		TestLabel:   "TestLabel",
		UsedNetwork: "TestNetwork",
		Env: stEnv{
			blockNumber: 1,
			Coinbase:    common.Address{},
			Difficulty:  &util.BigInt{*bInt},
			GasLimit:    &util.BigInt{*bInt},
			Number:      &util.BigInt{*bInt},
			Timestamp:   &util.BigInt{*bInt},
			BaseFee:     &util.BigInt{*bInt},
		},
		Pre: core.GenesisAlloc{
			common.HexToAddress("0x1"): core.GenesisAccount{
				Balance: big.NewInt(1000),
				Nonce:   1,
			},
			common.HexToAddress("0x2"): core.GenesisAccount{
				Balance: big.NewInt(2000),
				Nonce:   2,
			},
		},
		Tx: stTransaction{
			GasPrice:             &util.BigInt{*bInt},
			MaxFeePerGas:         &util.BigInt{*bInt},
			MaxPriorityFeePerGas: &util.BigInt{*bInt},
			Nonce:                &util.BigInt{*bInt},
			To:                   common.HexToAddress("0x10").Hex(),
			Data:                 []string{"0x"},
			GasLimit:             []*util.BigInt{{*bInt}},
			Value:                []string{"0x01"},
			PrivateKey:           hexutil.MustDecode("0x45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8"),
		},
		Post: map[string][]stPostState{
			"TestNetwork": {
				{
					RootHash: common.HexToHash("0x20"),
					LogsHash: common.HexToHash("0x30"),
					indexes:  Index{},
				},
			},
		},
		//Out: hexutil.Bytes("0x0"),
	}
}
