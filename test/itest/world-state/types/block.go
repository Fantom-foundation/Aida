package types

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/ethereum/go-ethereum/common"
)

// BlockEpochState encapsulates block processing and epoch states.
type BlockEpochState struct {
	BlockState *BlockState
	EpochState *EpochState
}

// BlockState represents a state of a block processing.
type BlockState struct {
	LastBlock             BlockCtx
	FinalizedStateRoot    common.Hash
	EpochGas              uint64
	EpochCheaters         lachesis.Cheaters
	CheatersWritten       uint32
	ValidatorStates       []ValidatorBlockState
	NextValidatorProfiles ValidatorProfiles
	DirtyRules            *Rules `rlp:"nil"` // nil means that there's no changes compared to epoch rules
	AdvanceEpochs         idx.Epoch
}

// BlockCtx holds basic information about a block context.
type BlockCtx struct {
	Idx     uint64
	Time    uint64
	Atropos common.Hash
}

// GasPowerLeft is long-term gas power left and short-term gas power left.
type GasPowerLeft struct {
	Gas [2]uint64
}

type (
	// Timestamp is a UNIX nanoseconds timestamp
	Timestamp uint64
)

type Block struct {
	Time        Timestamp
	Atropos     hash.Event
	Events      hash.Events
	Txs         []common.Hash // non event txs (received via genesis or LLR)
	InternalTxs []common.Hash // DEPRECATED in favor of using only Txs fields and method internal.IsInternal
	SkippedTxs  []uint32      // indexes of skipped txs, starting from first tx of first event, ending with last tx of last event
	GasUsed     uint64
	Root        hash.Hash
}
