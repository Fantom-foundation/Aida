package executor

//go:generate mockgen -source substate_provider.go -destination substate_provider_mocks.go -package executor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// ----------------------------------------------------------------------------
//                              Implementation
// ----------------------------------------------------------------------------

// OpenSubstateDb opens a substate database as configured in the given parameters.
func OpenSubstateDb(config *utils.Config, ctxt *cli.Context) (res Provider[*substate.Substate], err error) {
	// Substate is panicking if we are opening a non-existing directory. To mitigate
	// the damage, we recover here and forward an error instead.
	defer func() {
		if issue := recover(); issue != nil {
			res = nil
			err = fmt.Errorf("failed to open substate DB: %v", issue)
		}
	}()
	substate.SetSubstateDb(config.AidaDb)
	substate.OpenSubstateDBReadOnly()
	return &substateProvider{ctxt, config.Workers}, nil
}

// substateProvider is an adapter of Aida's SubstateProvider interface defined above to the
// current substate implementation offered by github.com/Fantom-foundation/Substate.
type substateProvider struct {
	ctxt                *cli.Context
	numParallelDecoders int
}

func (s substateProvider) Run(from int, to int, consumer Consumer[*substate.Substate]) error {
	iter := substate.NewSubstateIterator(uint64(from), s.numParallelDecoders)
	defer iter.Release()
	for iter.Next() {
		tx := iter.Value()
		if tx.Block >= uint64(to) {
			return nil
		}
		if err := consumer(TransactionInfo[*substate.Substate]{int(tx.Block), tx.Transaction, tx.Substate}); err != nil {
			return err
		}
	}
	return nil
}

func (substateProvider) Close() {
	substate.CloseSubstateDB()
}
