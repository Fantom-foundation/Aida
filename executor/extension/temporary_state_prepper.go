package extension

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
)

// MakeTemporaryStatePrepper creates an extension that introduces a fresh
// in-memory StateDB instance before each transaction execution.
func MakeTemporaryStatePrepper() executor.Extension {
	return temporaryStatePrepper{}
}

type temporaryStatePrepper struct {
	NilExtension
}

func (temporaryStatePrepper) PreTransaction(s executor.State, c *executor.Context) error {
	c.State = state.MakeInMemoryStateDB(&s.Substate.InputAlloc, uint64(s.Block))
	return nil
}
