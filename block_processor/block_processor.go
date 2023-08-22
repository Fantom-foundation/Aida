package blockprocessor

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// ProcessorExtensions supports block processing actions
type ProcessorExtensions interface {
	Init(*BlockProcessor) error            // Initialise action (before block processing starts)
	PostPrepare(*BlockProcessor) error     // Post-prepare action (after statedb has been created/primed)
	PostTransaction(*BlockProcessor) error // Post-transaction action (after a transaction has been processed)
	PostBlock(*BlockProcessor) error       // Post-block action (after a block has been processed)
	PostProcessing(*BlockProcessor) error  // Post-processing action (after all transactions have been processed/before closing statedb)
	Exit(*BlockProcessor) error            // Exit action (after completing block processing)
}

type ExtensionList []ProcessorExtensions

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
}

// NewBlockProcessor creates a new block processor instance
func NewBlockProcessor(name string, ctx *cli.Context) (*BlockProcessor, error) {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return nil, err
	}

	return &BlockProcessor{
		cfg:      cfg,
		totalGas: new(big.Int),
		log:      logger.NewLogger(cfg.LogLevel, name),
	}, nil
}

// ExecuteExtensions executes a matching method name of actions in the action list.
func (al ExtensionList) ExecuteExtensions(method string, bp *BlockProcessor) error {
	inputs := make([]reflect.Value, 1)
	inputs[0] = reflect.ValueOf(bp)

	for _, action := range al {
		out := reflect.ValueOf(action).MethodByName(method).Call(inputs)
		if out[0].Interface() != nil {
			return out[0].Interface().(error)
		}
	}
	return nil
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

// ProcessFirstBlock sets appropriate block and sync period number and
// process transaction.
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
	bp.totalTx = 0
	bp.db.BeginSyncPeriod(bp.syncPeriod)
	bp.db.BeginBlock(bp.block)

	// process transaction
	if err := utils.ProcessTx(bp.db, bp.cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
		bp.log.Criticalf("\tFailed processing transaction: %v", err)
		return err
	}
	bp.totalGas.Add(bp.totalGas, new(big.Int).SetUint64(tx.Substate.Result.GasUsed))
	return nil
}

// Run executes all blocks in sequence.
func (bp *BlockProcessor) Run(actions ExtensionList) error {
	var err error

	// reset state
	bp.block = uint64(0)
	bp.syncPeriod = uint64(0)

	// TODO: there should not be a side-effect on cfg in runvm - that is a design failure
	bp.cfg.StateValidationMode = utils.SubsetCheck

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

	// create new iterator over substates and iterate
	bp.log.Notice("Process blocks")
	iter := substate.NewSubstateIterator(bp.cfg.First, bp.cfg.Workers)
	defer iter.Release()

	// process the first block
	if err = bp.ProcessFirstBlock(iter); err != nil {
		return err
	}

	// process the remaining blocks
	if err = bp.iterate(iter, actions); err != nil {
		return err
	}

	bp.db.EndBlock()
	bp.db.EndSyncPeriod()

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

// iterate over substate
func (bp *BlockProcessor) iterate(iter substate.SubstateIterator, actions ExtensionList) error {
	var err error

	for iter.Next() {
		bp.tx = iter.Value()

		// initiate first sync-period and block.
		// close off old block and possibly sync-periods
		if bp.block != bp.tx.Block {
			// exit if we processed last block
			if bp.tx.Block > bp.cfg.Last {
				return nil
			}

			if err = actions.ExecuteExtensions("PostBlock", bp); err != nil {
				return err
			}
		}

		// check whether we have processed enough transaction
		// TODO: cfg.MaxNumTransactions should be a uint64 flag
		if bp.cfg.MaxNumTransactions >= 0 && bp.totalTx >= uint64(bp.cfg.MaxNumTransactions) {
			break
		}

		// process transaction
		if err = utils.ProcessTx(bp.db, bp.cfg, bp.tx.Block, bp.tx.Transaction, bp.tx.Substate); err != nil {
			bp.log.Criticalf("\tFailed processing transaction: %v", err)
			return err
		}

		bp.totalGas.Add(bp.totalGas, new(big.Int).SetUint64(bp.tx.Substate.Result.GasUsed))

		// call post-transaction actions
		if err = actions.ExecuteExtensions("PostTransaction", bp); err != nil {
			return err
		}

		bp.totalTx++
	}

	return nil
}

// GetConfig provides the processes configuration parsed by this block processor
// from command line parameters, default values, and other sources.
func (bp *BlockProcessor) GetConfig() *utils.Config {
	return bp.cfg
}
