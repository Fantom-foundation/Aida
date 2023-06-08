package types

import (
	"fmt"
	"io"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	berlinBit = 1 << 0
	londonBit = 1 << 1
	llrBit    = 1 << 2
)

// RulesRLP represents a set of network rules applicable for the given epoch and block processing.
type RulesRLP struct {
	Name      string
	NetworkID uint64

	// Graph options
	Dag DagRules

	// Epochs options
	Epochs EpochsRules

	// Blockchain options
	Blocks BlocksRules

	// Economy options
	Economy EconomyRules

	Upgrades Upgrades `rlp:"-"`
}

// Rules describes opera net.
// Note keep track of all the non-copiable variables in Copy()
type Rules RulesRLP

// DagRules of Lachesis DAG (directed acyclic graph).
type DagRules struct {
	MaxParents     idx.Event
	MaxFreeParents idx.Event // maximum number of parents with no gas cost
	MaxExtraData   uint32
}

// EpochsRules represents rules for epoch closing.
type EpochsRules struct {
	MaxEpochGas      uint64
	MaxEpochDuration uint64
}

// BlocksRules represents rules for blocks closing.
type BlocksRules struct {
	MaxBlockGas             uint64 // technical hard limit, gas is mostly governed by gas power allocation
	MaxEmptyBlockSkipPeriod uint64
}

// EconomyRules contains economy constants
type EconomyRules struct {
	BlockMissedSlack idx.Block
	Gas              GasRules
	MinGasPrice      *big.Int
	ShortGasPower    GasPowerRules
	LongGasPower     GasPowerRules
}

// GasRulesRLPV1 represents rules for applying gas into consensus operations; version 1.
type GasRulesRLPV1 struct {
	MaxEventGas  uint64
	EventGas     uint64
	ParentGas    uint64
	ExtraDataGas uint64
	// Post-LLR fields
	BlockVotesBaseGas    uint64
	BlockVoteGas         uint64
	EpochVoteGas         uint64
	MisbehaviourProofGas uint64
}

// GasRulesRLPV0 represents rules for applying gas into consensus operations; version 0.
type GasRulesRLPV0 struct {
	MaxEventGas  uint64
	EventGas     uint64
	ParentGas    uint64
	ExtraDataGas uint64
}

type GasRules GasRulesRLPV1

// GasPowerRules defines gas power rules in the consensus.
type GasPowerRules struct {
	AllocPerSec        uint64
	MaxAllocPeriod     uint64
	StartupAllocPeriod uint64
	MinStartupGas      uint64
}

// Upgrades represents set of flags for critical network behaviour upgrades.
type Upgrades struct {
	Berlin bool
	London bool
	Llr    bool
}

// DecodeRLP decodes Rules structure from the RLP stream.
func (r *Rules) DecodeRLP(s *rlp.Stream) error {
	kind, _, err := s.Kind()
	if err != nil {
		return err
	}

	// read rType
	rType := uint8(0)
	if kind == rlp.Byte {
		var b []byte
		if b, err = s.Bytes(); err != nil {
			return err
		}
		if len(b) == 0 {
			return fmt.Errorf("empty typed")
		}
		rType = b[0]
		if rType == 0 || rType > 1 {
			return fmt.Errorf("unknown type")
		}
	}

	// decode the main body
	rlpR := RulesRLP{}
	err = s.Decode(&rlpR)
	if err != nil {
		return err
	}
	*r = Rules(rlpR)

	// decode additional fields, depending on the type
	if rType >= 1 {
		err = s.Decode(&r.Upgrades)
		if err != nil {
			return err
		}
	}
	return nil
}

// EncodeRLP encodes Rules structure nto the RLP stream.
func (r *Rules) EncodeRLP(w io.Writer) error {
	// write the type
	rType := uint8(0)
	if r.Upgrades != (Upgrades{}) {
		rType = 1
		_, err := w.Write([]byte{rType})
		if err != nil {
			return err
		}
	}

	// write the main body
	rlpR := RulesRLP(*r)
	err := rlp.Encode(w, &rlpR)
	if err != nil {
		return err
	}

	// write additional fields, depending on the type
	if rType > 0 {
		err := rlp.Encode(w, &r.Upgrades)
		if err != nil {
			return err
		}
	}
	return nil
}

// DecodeRLP decodes Upgrades structure from RLP stream.
func (u *Upgrades) DecodeRLP(s *rlp.Stream) error {
	bitmap := struct {
		V uint64
	}{}
	err := s.Decode(&bitmap)
	if err != nil {
		return err
	}
	u.Berlin = (bitmap.V & berlinBit) != 0
	u.London = (bitmap.V & londonBit) != 0
	u.Llr = (bitmap.V & llrBit) != 0
	return nil
}

// EncodeRLP encodes Upgrades into the RLP stream.
func (u *Upgrades) EncodeRLP(w io.Writer) error {
	bitmap := struct {
		V uint64
	}{}
	if u.Berlin {
		bitmap.V |= berlinBit
	}
	if u.London {
		bitmap.V |= londonBit
	}
	if u.Llr {
		bitmap.V |= llrBit
	}
	return rlp.Encode(w, &bitmap)
}

// DecodeRLP decodes Gas Rules from the RLP stream.
func (r *GasRules) DecodeRLP(s *rlp.Stream) error {
	kind, _, err := s.Kind()
	if err != nil {
		return err
	}

	// read rType
	rType := uint8(0)
	if kind == rlp.Byte {
		var b []byte
		if b, err = s.Bytes(); err != nil {
			return err
		}
		if len(b) == 0 {
			return fmt.Errorf("empty typed")
		}
		rType = b[0]
		if rType == 0 || rType > 1 {
			return fmt.Errorf("unknown type")
		}
	}

	// decode the main body
	if rType == 0 {
		rlpR := GasRulesRLPV0{}
		err = s.Decode(&rlpR)
		if err != nil {
			return err
		}
		*r = GasRules{
			MaxEventGas:  rlpR.MaxEventGas,
			EventGas:     rlpR.EventGas,
			ParentGas:    rlpR.ParentGas,
			ExtraDataGas: rlpR.ExtraDataGas,
		}
		return nil
	} else {
		return s.Decode((*GasRulesRLPV1)(r))
	}
}

// EncodeRLP encodes Gas Rules into the RLP stream.
func (r *GasRules) EncodeRLP(w io.Writer) error {
	// write the type
	rType := uint8(0)
	if r.EpochVoteGas != 0 || r.MisbehaviourProofGas != 0 || r.BlockVotesBaseGas != 0 || r.BlockVoteGas != 0 {
		rType = 1
		_, err := w.Write([]byte{rType})
		if err != nil {
			return err
		}
	}
	if rType == 0 {
		return rlp.Encode(w, &GasRulesRLPV0{
			MaxEventGas:  r.MaxEventGas,
			EventGas:     r.EventGas,
			ParentGas:    r.ParentGas,
			ExtraDataGas: r.ExtraDataGas,
		})
	} else {
		return rlp.Encode(w, (*GasRulesRLPV1)(r))
	}
}
