package utils

import (
	"context"

	"github.com/Fantom-foundation/Aida/state"
	erigonethdb "github.com/ledgerwatch/erigon/ethdb"
)

func newBatchExecution(db state.StateDB, cfg *Config) erigonethdb.DbWithPendingMutations {
	batch := db.NewBatch(cfg.rwTx, cfg.QuitCh)
	db.BeginBlockApplyBatch(batch, false, cfg.rwTx)
	return batch
}

func BeginRwTxBatch(db state.StateDB, cfg *Config) (err error) {
	cfg.rwTx, err = db.DB().RwKV().BeginRw(context.Background())
	if err != nil {
		return err
	}

	// start erigon batch execution
	cfg.batch = newBatchExecution(db, cfg)
	return
}

func CommitBatchRwTx(cfg *Config) (err error) {
	err = cfg.batch.Commit()
	if err != nil {
		return
	}

	return cfg.rwTx.Commit()
}

func CommitAndBegin(db state.StateDB, cfg *Config) error {
	if err := CommitBatchRwTx(cfg); err != nil {
		return err
	}

	return BeginRwTxBatch(db, cfg)
}

func Rollback(cfg *Config) {
	cfg.rwTx.Rollback()
	cfg.batch.Rollback()
}
