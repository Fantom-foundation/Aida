package state

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestNewChainConduit(t *testing.T) {
	tests := []struct {
		isEthereum  bool
		chainConfig *params.ChainConfig
		want        *ChainConduit
	}{
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			want:        &ChainConduit{isEthereum: true, chainConfig: params.MainnetChainConfig},
		},
		{
			isEthereum:  false,
			chainConfig: nil,
			want:        &ChainConduit{isEthereum: false, chainConfig: nil},
		},
	}
	for _, test := range tests {
		got := NewChainConduit(test.isEthereum, test.chainConfig)
		gotS, err := json.Marshal(got)
		if err != nil {
			t.Errorf("json.Marshal(%v) failed: %v", got, err)
		}
		wantS, err := json.Marshal(test.want)
		if err != nil {
			t.Errorf("json.Marshal(%v) failed: %v", test.want, err)
		}
		if string(gotS) != string(wantS) {
			t.Errorf("NewChainConduit(%v, %v) = %v, want %v", test.isEthereum, test.chainConfig, got, test.want)
		}
	}
}

func TestChainConduit_IsFinalise(t *testing.T) {
	tests := []struct {
		isEthereum  bool
		chainConfig *params.ChainConfig
		block       *big.Int
		want        bool
	}{
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			block:       big.NewInt(2_674_999),
			want:        false,
		},
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			block:       big.NewInt(2_675_000),
			want:        false,
		},
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			block:       big.NewInt(4_369_999),
			want:        false,
		},
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			block:       big.NewInt(4_370_000),
			want:        true,
		},
		{
			isEthereum:  false,
			chainConfig: nil,
			block:       big.NewInt(1),
			want:        true,
		},
	}
	for _, test := range tests {
		c := NewChainConduit(test.isEthereum, test.chainConfig)
		got := c.IsFinalise(test.block)
		if got != test.want {
			t.Errorf("ChainConduit.IsFinalise(%v)[%v, %v] = %v, want %v", test.block, test.isEthereum, test.chainConfig, got, test.want)
		}
	}
}

func TestChainConduit_DeleteEmptyObjects(t *testing.T) {
	tests := []struct {
		isEthereum  bool
		chainConfig *params.ChainConfig
		block       *big.Int
		want        bool
	}{
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			block:       big.NewInt(2_674_999),
			want:        false,
		},
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			block:       big.NewInt(2_675_000),
			want:        true,
		},
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			block:       big.NewInt(4_369_999),
			want:        true,
		},
		{
			isEthereum:  true,
			chainConfig: params.MainnetChainConfig,
			block:       big.NewInt(4_370_000),
			want:        true,
		},
		{
			isEthereum:  false,
			chainConfig: nil,
			block:       big.NewInt(1),
			want:        true,
		},
	}
	for _, test := range tests {
		c := NewChainConduit(test.isEthereum, test.chainConfig)
		got := c.DeleteEmptyObjects(test.block)
		if got != test.want {
			t.Errorf("ChainConduit.DeleteEmptyObjects(%v)[%v, %v] = %v, want %v", test.block, test.isEthereum, test.chainConfig, got, test.want)
		}
	}
}
