package txcontext

import (
	"github.com/ethereum/go-ethereum/core"
)

// NilTxContext is an implementation of the TxContext interface
// ignoring all incoming events. It is mainly intended as a fall-back
// implementation when a no-op implementation is required, as well as
// an implementation that may be embedded in other extensions to avoid
// the need to implement all possible events.
type NilTxContext struct{}

func (NilTxContext) GetInputState() WorldState { return nil }

func (NilTxContext) GetBlockEnvironment() BlockEnvironment { return nil }

func (NilTxContext) GetMessage() core.Message { return nil }

func (NilTxContext) GetOutputState() WorldState { return nil }

func (NilTxContext) GetResult() Result { return nil }
