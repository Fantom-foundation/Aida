package types

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
)

// EpochState represents the state of epoch processing.
type EpochState struct {
	Epoch             idx.Epoch
	EpochStart        uint64
	PrevEpochStart    uint64
	EpochStateRoot    common.Hash
	Validators        *pos.Validators
	ValidatorStates   []ValidatorEpochState
	ValidatorProfiles ValidatorProfiles
	Rules             Rules
}
