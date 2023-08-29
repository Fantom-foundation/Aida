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

	actions := blockprocessor.NewExtensionList([]blockprocessor.ProcessorExtensions{
		blockprocessor.NewProgressReportExtension(cfg),
		blockprocessor.NewValidationExtension(),
		blockprocessor.NewProfileExtension(),
		blockprocessor.NewDbManagerExtension(),
		blockprocessor.NewProxyLoggerExtension(),
		blockprocessor.NewProxyProfilerExtension(),
	})

	bp := NewVmSdb(cfg, actions)
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

func (bp *VmSdb) Run() error {
	var err error

	// TODO: there should not be a side-effect on cfg in runvm - that is a design failure
	bp.Cfg.StateValidationMode = utils.SubsetCheck
	bp.Cfg.CopySrcDb = true

	// call init actions
	if err = bp.ExecuteExtension("Init"); err != nil {
		return err
	}

	// close actions when return
	defer func() error {
		return bp.Exit()
	}()

	// prepare statedb and priming
	if err = bp.Prepare(); err != nil {
		return fmt.Errorf("cannot prepare block processor; %v", err)
	}

	// create new BasicIterator over substates and BasicIterator
	bp.Log.Notice("Process blocks")
	iter := substate.NewSubstateIterator(bp.Cfg.First, bp.Cfg.Workers)
	defer iter.Release()

	if err = bp.ProcessFirstBlock(iter); err != nil {
		return err
	}

	// process the remaining blocks
	if err = bp.Iterate(iter); err != nil {
		return err
	}

	bp.Db.EndBlock()
	bp.Db.EndSyncPeriod()
	bp.Log.Noticef("%v errors found.", utils.NumErrors)

	// call post-processing actions
	if err = bp.ExecuteExtension("PostProcessing"); err != nil {
		return err
	}

	// close the DB and print disk usage
	bp.Log.Info("Close StateDB")
	if err = bp.Db.Close(); err != nil {
		return fmt.Errorf("Failed to close database: %v", err)
	}

	return err
}

// ProcessFirstBlock sets appropriate block and sync period number and process transaction.
func (bp *VmSdb) ProcessFirstBlock(iter substate.SubstateIterator) error {
	// no transaction available for the specified range
	if !iter.Next() {
		return nil
	}

	// process first transaction
	tx := iter.Value()
	if tx.Block > bp.Cfg.Last {
		return nil
	}
	bp.syncPeriod = tx.Block / bp.Cfg.SyncPeriodLength
	bp.Block = tx.Block
	bp.Db.BeginSyncPeriod(bp.syncPeriod)
	bp.Db.BeginBlock(bp.Block)

	// process transaction
	if _, err := utils.ProcessTx(bp.Db, bp.Cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
		bp.Log.Criticalf("\tFailed processing transaction: %v", err)
		return err
	}

	bp.TotalTx.SetUint64(bp.TotalTx.Uint64() + tx.Substate.Result.GasUsed)
	return nil
}

// Iterate over substates
func (bp *VmSdb) Iterate(iter substate.SubstateIterator) error {
	var (
		err           error
		newSyncPeriod uint64
		txsInBlock    uint64
		gasInBlock    uint64
	)

	for iter.Next() {
		bp.tx = iter.Value()

		// initiate first sync-period and block.
		// close off old block and possibly sync-periods
		if bp.Block != bp.tx.Block {
			// exit if we processed last block
			if bp.tx.Block > bp.Cfg.Last {
				return nil
			}

			bp.Db.EndBlock()

			// add txs and gas for the block
			bp.TotalTx.SetUint64(bp.TotalTx.Uint64() + txsInBlock)
			bp.TotalGas.SetUint64(bp.TotalGas.Uint64() + gasInBlock)

			txsInBlock = 0
			gasInBlock = 0

			if err = bp.ExecuteExtension("PostBlock"); err != nil {
				return err
			}

			// switch to next sync-period if needed.
			// TODO: Revisit semantics - is this really necessary ????
			newSyncPeriod = bp.tx.Block / bp.Cfg.SyncPeriodLength
			for bp.syncPeriod < newSyncPeriod {
				bp.Db.EndSyncPeriod()
				bp.syncPeriod++
				bp.Db.BeginSyncPeriod(bp.syncPeriod)
			}

			bp.Block = bp.tx.Block
			bp.Db.BeginBlock(bp.Block)

		}

		// check whether we have processed enough transaction
		// TODO: cfg.MaxNumTransactions should be a uint64 flag
		if bp.Cfg.MaxNumTransactions >= 0 && bp.TotalTx.Uint64() >= uint64(bp.Cfg.MaxNumTransactions) {
			break
		}

		// process transaction
		if _, err = utils.ProcessTx(bp.Db, bp.Cfg, bp.tx.Block, bp.tx.Transaction, bp.tx.Substate); err != nil {
			bp.Log.Criticalf("\tFailed processing transaction: %v", err)
			return err
		}

		// call post-transaction actions
		if err = bp.ExecuteExtension("PostTransaction"); err != nil {
			return err
		}

		txsInBlock++
		gasInBlock += bp.tx.Substate.Result.GasUsed
	}
	return nil
}
