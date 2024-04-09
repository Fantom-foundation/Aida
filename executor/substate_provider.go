package executor

//go:generate mockgen -source substate_provider.go -destination substate_provider_mocks.go -package executor

import (
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/urfave/cli/v2"
)

// ----------------------------------------------------------------------------
//                              Implementation
// ----------------------------------------------------------------------------

// OpenSubstateProvider opens a substate database as configured in the given parameters.
func OpenSubstateProvider(cfg *utils.Config, ctxt *cli.Context, aidaDb db.BaseDB) Provider[txcontext.TxContext] {
	substateDb := db.MakeDefaultSubstateDBFromBaseDB(aidaDb)
	return &substateProvider{substateDb, ctxt, cfg.Workers}
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
		if err := consumer(TransactionInfo[txcontext.TxContext]{int(tx.Block), tx.Transaction, substatecontext.NewTxContext(tx)}); err != nil {
			return err
		}
	}
	// this cannot be used in defer because Release() has a WaitGroup.Wait() call
	// so if called after iter.Error() there is a change the error does not get distributed.
	iter.Release()
	return iter.Error()
}

func (s substateProvider) Close() {
	// ignored, database is opened it top-most level
}
