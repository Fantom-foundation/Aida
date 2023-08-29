package vm_sdb

import (
	"fmt"

	blockprocessor "github.com/Fantom-foundation/Aida/block_processor"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RunVmSdb performs block processing on a StateDb
func RunVmSdb(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	extensions := blockprocessor.NewExtensionList([]blockprocessor.ProcessorExtensions{
		blockprocessor.NewProgressReportExtension(),
		blockprocessor.NewValidationExtension(),
		blockprocessor.NewProfileExtension(),
		blockprocessor.NewDbManagerExtension(),
		blockprocessor.NewProxyLoggerExtension(),
	})

	bp := NewVmSdb(cfg, extensions)
	return bp.Run()
}

type VmSdb struct {
	*blockprocessor.BlockProcessor
	tx         *substate.Transaction // current tx
	syncPeriod uint64                // current sync period

}

// NewVmSdb returns new BlockProcessor for VmSdb
func NewVmSdb(cfg *utils.Config, actions blockprocessor.ExtensionList) *VmSdb {
	return &VmSdb{
		BlockProcessor: blockprocessor.NewBlockProcessor(cfg, actions, "Aida VM SDb"),
		syncPeriod:     0,
	}
}

func (sdb *VmSdb) Run() error {
	var err error

	// TODO: there should not be a side-effect on cfg in runvm - that is a design failure
	sdb.Cfg.StateValidationMode = utils.SubsetCheck
	sdb.Cfg.CopySrcDb = true

	// call init actions
	if err = sdb.ExecuteExtension("Init"); err != nil {
		return err
	}

	// close actions when return
	defer func() error {
		return sdb.Exit()
	}()

	// prepare statedb and priming
	if err = sdb.Prepare(); err != nil {
		return fmt.Errorf("cannot prepare block processor; %v", err)
	}

	// create new BasicIterator over substates and BasicIterator
	sdb.Log.Notice("Process blocks")
	iter := substate.NewSubstateIterator(sdb.Cfg.First, sdb.Cfg.Workers)
	defer iter.Release()

	if err = sdb.ProcessFirstBlock(iter); err != nil {
		return err
	}

	// process the remaining blocks
	if err = sdb.Iterate(iter); err != nil {
		return err
	}

	sdb.Db.EndBlock()
	sdb.Db.EndSyncPeriod()
	sdb.Log.Noticef("%v errors found.", utils.NumErrors)

	// call post-processing actions
	if err = sdb.ExecuteExtension("PostProcessing"); err != nil {
		return err
	}

	// close the DB and print disk usage
	sdb.Log.Info("Close StateDB")
	if err = sdb.Db.Close(); err != nil {
		return fmt.Errorf("Failed to close database: %v", err)
	}

	return err
}

// ProcessFirstBlock sets appropriate block and sync period number and process transaction.
func (sdb *VmSdb) ProcessFirstBlock(iter substate.SubstateIterator) error {
	// no transaction available for the specified range
	if !iter.Next() {
		return nil
	}

	// process first transaction
	tx := iter.Value()
	if tx.Block > sdb.Cfg.Last {
		return nil
	}
	sdb.syncPeriod = tx.Block / sdb.Cfg.SyncPeriodLength
	sdb.Block = tx.Block
	sdb.Db.BeginSyncPeriod(sdb.syncPeriod)
	sdb.Db.BeginBlock(sdb.Block)

	// process transaction
	if _, err := utils.ProcessTx(sdb.Db, sdb.Cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
		sdb.Log.Criticalf("\tFailed processing transaction: %v", err)
		return err
	}

	sdb.TotalTx.SetUint64(sdb.TotalTx.Uint64() + tx.Substate.Result.GasUsed)
	return nil
}

// Iterate over substates
func (sdb *VmSdb) Iterate(iter substate.SubstateIterator) error {
	var (
		err           error
		newSyncPeriod uint64
		txsInBlock    uint64
		gasInBlock    uint64
	)

	for iter.Next() {
		sdb.tx = iter.Value()

		// initiate first sync-period and block.
		// close off old block and possibly sync-periods
		if sdb.Block != sdb.tx.Block {
			// exit if we processed last block
			if sdb.tx.Block > sdb.Cfg.Last {
				return nil
			}

			sdb.Db.EndBlock()

			// add txs and gas for the block
			sdb.TotalTx.SetUint64(sdb.TotalTx.Uint64() + txsInBlock)
			sdb.TotalGas.SetUint64(sdb.TotalGas.Uint64() + gasInBlock)

			if err = bp.ExecuteExtension("PostBlock"); err != nil {
				return err
			}

			txsInBlock = 0
			gasInBlock = 0

			// switch to next sync-period if needed.
			// TODO: Revisit semantics - is this really necessary ????
			newSyncPeriod = sdb.tx.Block / sdb.Cfg.SyncPeriodLength
			for sdb.syncPeriod < newSyncPeriod {
				sdb.Db.EndSyncPeriod()
				sdb.syncPeriod++
				sdb.Db.BeginSyncPeriod(sdb.syncPeriod)
			}

			sdb.Block = sdb.tx.Block
			sdb.Db.BeginBlock(sdb.Block)

		}

		// check whether we have processed enough transaction
		// TODO: cfg.MaxNumTransactions should be a uint64 flag
		if sdb.Cfg.MaxNumTransactions >= 0 && sdb.TotalTx.Uint64() >= uint64(sdb.Cfg.MaxNumTransactions) {
			break
		}

		// process transaction
		if _, err = utils.ProcessTx(sdb.Db, sdb.Cfg, sdb.tx.Block, sdb.tx.Transaction, sdb.tx.Substate); err != nil {
			sdb.Log.Criticalf("\tFailed processing transaction: %v", err)
			return err
		}

		// call post-transaction actions
		if err = sdb.ExecuteExtension("PostTransaction"); err != nil {
			return err
		}

		txsInBlock++
		gasInBlock += sdb.tx.Substate.Result.GasUsed
	}
	return nil
}
