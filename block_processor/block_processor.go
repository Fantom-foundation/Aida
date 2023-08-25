package blockprocessor

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

type BlockProcessor struct {
	cfg        *utils.Config         // configuration
	log        *logging.Logger       // logger
	stateDbDir string                // directory of the StateDB
	db         state.StateDB         // StateDB
	tx         *substate.Transaction // current tx
	block      uint64                // current block
	syncPeriod uint64                // current sync period
	totalTx    uint64                // total number of transactions so far
	totalGas   *big.Int              // total gas consumed so far

	// this is needed because some functionality needs to be executed only on vm-sdb, this will be removed in next refactor PR
	toolName string
}

const (
	VmSdbToolName = "vm-sdb"
	VmAdbToolName = "vm-adb"
)

// IterateFunc declares how iteration should be done
type IterateFunc func(substate.SubstateIterator, ExtensionList, *BlockProcessor) error

// NewBlockProcessor creates a new block processor instance
func NewBlockProcessor(ctx *cli.Context, name string) (*BlockProcessor, error) {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return nil, err
	}

	return &BlockProcessor{
		cfg:      cfg,
		totalGas: new(big.Int),
		log:      logger.NewLogger(cfg.LogLevel, name),
		toolName: name,
	}, nil
}

// Prepare creates and primes a stateDB.
func (bp *BlockProcessor) Prepare() error {
	var err error

	bp.log.Notice("Open StateDb")
	bp.db, bp.stateDbDir, err = utils.PrepareStateDB(bp.cfg)
	if err != nil {
		return err
	}

	if !bp.cfg.SkipPriming && bp.cfg.StateDbSrc == "" {
		if err := utils.LoadWorldStateAndPrime(bp.db, bp.cfg, bp.cfg.First-1); err != nil {
			return fmt.Errorf("priming failed. %v", err)
		}
	}
	return nil
}

// ProcessFirstBlock sets appropriate block and sync period number and process transaction.
func (bp *BlockProcessor) ProcessFirstBlock(iter substate.SubstateIterator) error {
	// no transaction available for the specified range
	if !iter.Next() {
		return nil
	}

	// process first transaction
	tx := iter.Value()
	if tx.Block > bp.cfg.Last {
		return nil
	}
	bp.syncPeriod = tx.Block / bp.cfg.SyncPeriodLength
	bp.block = tx.Block
	bp.db.BeginSyncPeriod(bp.syncPeriod)
	bp.db.BeginBlock(bp.block)

	// process transaction
	if _, err := utils.ProcessTx(bp.db, bp.cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
		bp.log.Criticalf("\tFailed processing transaction: %v", err)
		return err
	}
	bp.totalGas.Add(bp.totalGas, new(big.Int).SetUint64(tx.Substate.Result.GasUsed))
	return nil
}

// Run executes all blocks in sequence.
func (bp *BlockProcessor) Run(actions ExtensionList, iterate IterateFunc) error {
	var err error

	// reset state
	bp.block = uint64(0)
	bp.syncPeriod = uint64(0)

	// TODO: there should not be a side-effect on cfg in runvm - that is a design failure
	bp.cfg.StateValidationMode = utils.SubsetCheck
	// TODO: add this option back when splitting vm-adb's and vm-sdb's run func
	// bp.cfg.CopySrcDb = true

	// call init actions
	if err = actions.ExecuteExtensions("Init", bp); err != nil {
		return err
	}

	// close actions when return
	defer func() error {
		return actions.ExecuteExtensions("Exit", bp)
	}()

	// open substate database
	bp.log.Notice("Open substate database")
	substate.SetSubstateDb(bp.cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// prepare statedb and priming
	if err = bp.Prepare(); err != nil {
		return err
	}

	// call post-prepare actions
	if err = actions.ExecuteExtensions("PostPrepare", bp); err != nil {
		return err
	}

	// create new BasicIterator over substates and BasicIterator
	bp.log.Notice("Process blocks")
	iter := substate.NewSubstateIterator(bp.cfg.First, bp.cfg.Workers)
	defer iter.Release()

	if bp.toolName == VmSdbToolName {
		// process the first block
		if err = bp.ProcessFirstBlock(iter); err != nil {
			return err
		}
	}

	// process the remaining blocks
	if err = iterate(iter, actions, bp); err != nil {
		return err
	}
	if bp.toolName == VmSdbToolName {
		bp.db.EndBlock()
		bp.db.EndSyncPeriod()
	}
	bp.log.Noticef("%v errors found.", utils.NumErrors)

	// call post-processing actions
	if err = actions.ExecuteExtensions("PostProcessing", bp); err != nil {
		return err
	}

	// close the DB and print disk usage
	bp.log.Info("Close StateDB")
	if err := bp.db.Close(); err != nil {
		return fmt.Errorf("Failed to close database: %v", err)
	}

	return err
}

// GetConfig provides the processes configuration parsed by this block processor
// from command line parameters, default values, and other sources.
func (bp *BlockProcessor) GetConfig() *utils.Config {
	return bp.cfg
}
