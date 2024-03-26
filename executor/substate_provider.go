package executor

//go:generate mockgen -source substate_provider.go -destination substate_provider_mocks.go -package executor

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/txcontext/substate/newsubstate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/urfave/cli/v2"
)

// ----------------------------------------------------------------------------
//                              Implementation
// ----------------------------------------------------------------------------

// OpenSubstateDb opens a substate database as configured in the given parameters.
func OpenSubstateDb(cfg *utils.Config, ctxt *cli.Context) (res Provider[txcontext.TxContext], err error) {
	db, err := db.NewDefaultSubstateDB(cfg.AidaDb)
	if err != nil {
		return nil, err
	}

	return &substateProvider{db, ctxt, cfg.Workers}, nil
}

// substateProvider is an adapter of Aida's SubstateProvider interface defined above to the
// current substate implementation offered by github.com/Fantom-foundation/Substate.
type substateProvider struct {
	db                  db.SubstateDB
	ctxt                *cli.Context
	numParallelDecoders int
}

func (s substateProvider) Run(from int, to int, consumer Consumer[txcontext.TxContext]) error {
	iter := s.db.NewSubstateIterator(from, s.numParallelDecoders)
	for iter.Next() {
		tx := iter.Value()
		if tx.Block >= uint64(to) {
			return nil
		}
		if err := consumer(TransactionInfo[txcontext.TxContext]{int(tx.Block), tx.Transaction, newsubstate.NewTxContext(tx)}); err != nil {
			return err
		}
	}
	// this cannot be used in defer because Release() has a WaitGroup.Wait() call
	// so if called after iter.Error() there is a change the error does not get distributed.
	iter.Release()
	return iter.Error()
}

func (s substateProvider) Close() {
	s.db.Close()
}
