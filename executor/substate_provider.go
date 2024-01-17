package executor

//go:generate mockgen -source substate_provider.go -destination substate_provider_mocks.go -package executor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor/transaction"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/urfave/cli/v2"
)

// ----------------------------------------------------------------------------
//                              Implementation
// ----------------------------------------------------------------------------

// OpenSubstateDb opens a substate database as configured in the given parameters.
func OpenSubstateDb(cfg *utils.Config, ctxt *cli.Context) (res Provider[transaction.SubstateData], err error) {
	// Substate is panicking if we are opening a non-existing directory. To mitigate
	// the damage, we recover here and forward an error instead.
	defer func() {
		if issue := recover(); issue != nil {
			res = nil
			err = fmt.Errorf("failed to open substate DB: %v", issue)
		}
	}()
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

func (s substateProvider) Run(from int, to int, consumer Consumer[transaction.SubstateData]) error {
	iter := s.db.NewSubstateIterator(from, s.numParallelDecoders)
	for iter.Next() {
		tx := iter.Value()
		if tx.Block >= uint64(to) {
			return nil
		}
		if err := consumer(TransactionInfo[transaction.SubstateData]{int(tx.Block), tx.Transaction, transaction.NewSubstateData(tx)}); err != nil {
			return err
		}
	}
	iter.Release()

	return iter.Error()
}

func (s substateProvider) Close() {
	s.db.Close()
}
