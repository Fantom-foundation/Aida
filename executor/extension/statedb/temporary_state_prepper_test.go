package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
)

func TestTemporaryStatePrepper_DefaultDbVariantIsOffTheChainStateDb(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbVariant = ""

	ext := MakeTemporaryStatePrepper(cfg)

	if _, ok := ext.(temporaryOffTheChainStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}
}

func TestTemporaryStatePrepper_OffTheChainDbVariant(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbVariant = "off-the-chain"

	ext := MakeTemporaryStatePrepper(cfg)

	if _, ok := ext.(temporaryOffTheChainStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}

}

func TestTemporaryStatePrepper_InMemoryDbVariant(t *testing.T) {
	cfg := &utils.Config{}
	cfg.DbVariant = "in-memory"

	ext := MakeTemporaryStatePrepper(cfg)

	if _, ok := ext.(temporaryInMemoryStatePrepper); !ok {
		t.Fatal("unexpected extension type")
	}
}
