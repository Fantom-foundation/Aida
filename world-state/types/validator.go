package types

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
)

// ValidatorBlockState represents the state of a validator on the block processing.
type ValidatorBlockState struct {
	LastEvent        EventInfo
	Uptime           uint64
	LastOnlineTime   uint64
	LastGasPowerLeft GasPowerLeft
	LastBlock        idx.Block
	DirtyGasRefund   uint64
	Originated       *big.Int
}

// ValidatorProfiles is a map between validator ID and validator information.
type ValidatorProfiles map[idx.ValidatorID]Validator

// Validator represents an information about specific validator in the consensus.
type Validator struct {
	Weight *big.Int
	PubKey PubKey
}

// PubKey encapsulates validator public key for consensus events verification.
type PubKey struct {
	Type uint8
	Raw  []byte
}

// ValidatorEpochState represents the state of a validator in epoch processing.
type ValidatorEpochState struct {
	GasRefund      uint64
	PrevEpochEvent EventInfo
}

// ValidatorAndID is pair Validator + ValidatorID
type ValidatorAndID struct {
	ValidatorID idx.ValidatorID
	Validator   Validator
}

// DecodeRLP decodes validator profiles from the input RLP stream.
func (vp *ValidatorProfiles) DecodeRLP(s *rlp.Stream) error {
	// decode into a simple array
	var arr []ValidatorAndID
	if err := s.Decode(&arr); err != nil {
		return err
	}

	// restore into the map
	*vp = make(ValidatorProfiles, len(arr))
	for _, it := range arr {
		(*vp)[it.ValidatorID] = it.Validator
	}

	return nil
}
