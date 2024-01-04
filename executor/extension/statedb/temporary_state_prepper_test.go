package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
)

func TestTemporaryStatePrepper_DefaultDbImplementationIsOffTheChainStateDb(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = ""

	ext := MakeTemporaryStatePrepper(cfg)

	// check that temporaryOffTheChainStatePrepper is default
	if _, ok := ext.(*temporaryOffTheChainStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}
}

func TestTemporaryStatePrepper_OffTheChainDbImplementation(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = "off-the-chain"

	ext := MakeTemporaryStatePrepper(cfg)

	if _, ok := ext.(*temporaryOffTheChainStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}

}

func TestTemporaryStatePrepper_InMemoryDbImplementation(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbImpl = "in-memory"

	ext := MakeTemporaryStatePrepper(cfg)

	if _, ok := ext.(temporaryInMemoryStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}
}
