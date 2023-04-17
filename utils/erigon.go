package utils

import (
	"context"

	"github.com/Fantom-foundation/Aida/state"

	"github.com/ledgerwatch/erigon-lib/kv"

	erigonethdb "github.com/ledgerwatch/erigon/ethdb"
	"github.com/ledgerwatch/erigon/ethdb/olddb"

	lru "github.com/hashicorp/golang-lru"
)

// newBatch begins new batch
func newBatch(cfg *Config) erigonethdb.DbWithPendingMutations {
	const lruDefaultSize = 1_000_000 // 56 MB

	whitelistedTables := []string{kv.Code, kv.ContractCode}

	contractCodeCache, err := lru.New(lruDefaultSize)
	if err != nil {
		panic(err)
	}

	// Contract code is unlikely to change too much, so let's keep it cached
	return olddb.NewHashBatch(cfg.rwTx, cfg.QuitCh, cfg.workingDirectory, whitelistedTables, contractCodeCache)
}

// newBatchExecution begins new batch and applies it to StateDB
func newBatchExecution(db state.StateDB, cfg *Config) erigonethdb.DbWithPendingMutations {
	batch := newBatch(cfg)
	db.BeginBlockApplyBatch(batch, cfg.rwTx)
	return batch
}

// BeginRwTxBatch begins erigon read/write transaction and batch
func BeginRwTxBatch(db state.StateDB, cfg *Config) (err error) {
	cfg.rwTx, err = db.DB().RwKV().BeginRw(context.Background())
	if err != nil {
		return err
	}

	// start erigon batch execution
	cfg.batch = newBatchExecution(db, cfg)
	return
}

// CommitBatchRwTx commits batch and erigon transaction
func CommitBatchRwTx(cfg *Config) (err error) {
	err = cfg.batch.Commit()
	if err != nil {
		return
	}

	return cfg.rwTx.Commit()
}

// CommitAndBegin commits batch and erigon transaction. It also begins new erigon transaction and batch
func CommitAndBegin(db state.StateDB, cfg *Config) error {
	if err := CommitBatchRwTx(cfg); err != nil {
		return err
	}

	return BeginRwTxBatch(db, cfg)
}

// Rollback rollbacks erigon transaction and batch
func Rollback(cfg *Config) {
	cfg.rwTx.Rollback()
	cfg.batch.Rollback()
}
