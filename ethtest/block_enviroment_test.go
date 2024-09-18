package ethtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	etests "github.com/ethereum/go-ethereum/tests"
)

func TestStBlockEnvironment_GetBaseFee(t *testing.T) {
	baseFee := newBigInt(10)

	tests := []struct {
		name     string
		baseFee  *BigInt
		want     *big.Int
		fork     string
		chainCfg *params.ChainConfig
	}{
		{
			name:    "Use_Predefined_If_nil",
			baseFee: nil,
			fork:    "London",
			want:    big.NewInt(0x0a),
		},
		{
			name:    "Pre_London_Returns_nil",
			baseFee: baseFee,
			fork:    "Berlin",
			want:    nil,
		},
		{
			name:    "Post_London_Uses_Given",
			baseFee: baseFee,
			fork:    "London",
			want:    baseFee.Convert(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chainCfg, _, err := etests.GetChainConfig(test.fork)
			if err != nil {
				t.Fatalf("cannot get chain config: %v", err)
			}
			env := stBlockEnvironment{BaseFee: baseFee, chainCfg: chainCfg}
			if got, want := env.GetBaseFee(), test.want; got.Cmp(want) != 0 {
				t.Errorf("unexpected base fee\ngot: %d\nwant: %d", got.Uint64(), want.Uint64())
			}
		})
	}
}

func TestStBlockEnvironment_GetBlockHash_Correctly_Converts(t *testing.T) {
	blockNum := uint64(10)
	want := common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(blockNum)).String())))
	env := &stBlockEnvironment{blockNumber: blockNum}

	got, err := env.GetBlockHash(blockNum)
	if err != nil {
		t.Fatalf("cannot get block hash: %v", err)
	}
	if want.Cmp(got) != 0 {
		t.Errorf("unexpected block hash, got: %s, want: %s", got, want)
	}
}

func TestStBlockEnvironment_GetGasLimit(t *testing.T) {
	tests := []struct {
		name     string
		gasLimit int64
		want     uint64
	}{
		{
			name:     "0_Uses_GenesisGasLimit",
			gasLimit: 0,
			want:     params.GenesisGasLimit,
		},
		{
			name:     "Non_0_Converts",
			gasLimit: 10,
			want:     10,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := &stBlockEnvironment{GasLimit: newBigInt(test.gasLimit)}
			if got, want := env.GetGasLimit(), test.want; got != want {
				t.Errorf("incorrect gas limit, got: %v, want: %v", got, want)
			}
		})
	}
}

func TestStBlockEnvironment_GetDifficulty(t *testing.T) {
	tests := []struct {
		name       string
		difficulty int64
		fork       string
		random     *BigInt
		want       uint64
	}{
		{
			name:       "PreLondon_Uses_Given",
			difficulty: 1,
			fork:       "Berlin",
			want:       1,
		},
		{
			name:       "PostLondon_With_NotNil_Random_Resets",
			difficulty: 1,
			fork:       "London",
			random:     newBigInt(1),
			want:       0,
		},
		{
			name:       "PostLondon_With_Nil_Random_Uses_Given",
			difficulty: 1,
			fork:       "London",
			want:       1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chainCfg, _, err := etests.GetChainConfig(test.fork)
			if err != nil {
				t.Fatalf("cannot get chain config: %v", err)
			}
			env := &stBlockEnvironment{
				Difficulty: newBigInt(test.difficulty),
				Random:     test.random,
				chainCfg:   chainCfg,
			}
			if got, want := env.GetDifficulty(), test.want; got.Uint64() != want {
				t.Errorf("incorrect gas limit, got: %v, want: %v", got, want)
			}
		})
	}
}

func TestStBlockEnvironment_GetBlobBaseFee(t *testing.T) {
	tests := []struct {
		name        string
		blobBaseFee int64
		fork        string
		want        *big.Int
	}{
		{
			name:        "PreCancun_Returns_Nil",
			blobBaseFee: 1,
			fork:        "London",
			want:        nil,
		},
		{
			name:        "PostCancun_Calculates",
			blobBaseFee: 1,
			fork:        "Cancun",
			want:        eip4844.CalcBlobFee(1),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chainCfg, _, err := etests.GetChainConfig(test.fork)
			if err != nil {
				t.Fatalf("cannot get chain config: %v", err)
			}
			env := &stBlockEnvironment{
				ExcessBlobGas: newBigInt(test.blobBaseFee),
				chainCfg:      chainCfg,
				Timestamp:     newBigInt(1),
			}

			got, want := env.GetBlobBaseFee(), test.want
			if got == nil && want == nil {
				return
			}
			if got.Cmp(want) == 0 {
				return
			}

			t.Errorf("incorrect gas limit, got: %d, want: %d", got, want)
		})
	}
}
