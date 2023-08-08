package vm_sdb

import (
	"fmt"
	"math/big"
	"os"
	"reflect"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// ProcessorActions supports block processing actions
type ProcessorActions interface {
	Init(*BlockProcessor) error            // Initialise action (before block processing starts)
	PostPrepare(*BlockProcessor) error     // Post-prepare action (after statedb has been created/primed)
	PostTransaction(*BlockProcessor) error // Post-transaction action (after a transaction has been processed)
	PostProcessing(*BlockProcessor) error  // Post-processing action (after all transactions have been processed/before closing statedb)
	Exit(*BlockProcessor) error            // Exit action (after completing block processing)
}

// BlockProcessor's state
type BlockProcessor struct {
	cfg        *utils.Config   // configuration
	log        *logging.Logger // logger
	stateDbDir string          // directory of the StateDB
	db         state.StateDB   // StateDB
	block      uint64          // current block
	syncPeriod uint64          // current sync period
	totalTx    uint64          // total number of transactions so far
	totalGas   *big.Int        // total gas consumed so far
}

// NewBlockProcessor creates a new block processor instance
func NewBlockProcessor(ctx *cli.Context) (*BlockProcessor, error) {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return nil, err
	}
	return &BlockProcessor{
		cfg:      cfg,
		totalGas: new(big.Int),
	}, nil
}

type ActionList []ProcessorActions

func (al ActionList) ExecuteAll(method string, args ...interface{}) error {
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}

	for _, action := range al {
		out := reflect.ValueOf(action).MethodByName(method).Call(inputs)
		if out[0].Interface() != nil {
			return out[0].Interface().(error)
		}
	}
	return nil
}

// Prepare creates and primes a stateDB
func (bp *BlockProcessor) Prepare() error {
	var err error
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

// Run executes all blocks in sequence
func (bp *BlockProcessor) Run(name string, actions ActionList) error {
	var err error

	// reset state
	bp.block = uint64(0)
	bp.syncPeriod = uint64(0)

	// TODO: there should not be a side-effect on cfg in runvm - that is a design failure
	bp.cfg.StateValidationMode = utils.SubsetCheck
	bp.cfg.CopySrcDb = true

	// retrieve configuration
	cfg := bp.cfg

	// open logger
	bp.log = logger.NewLogger(cfg.LogLevel, name)
	log := bp.log

	// call init actions
	if err := actions.ExecuteAll("Init", bp); err != nil {
		return err
	}

	defer func() error {
		return actions.ExecuteAll("Exit", bp)
	}()

	// open substate database
	log.Notice("Open substate database")
	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	if err := bp.Prepare(); err != nil {
		return err
	}
	if !cfg.KeepDb {
		log.Warningf("--keep-db is not used. Directory %v with DB will be removed at the end of this run.", bp.stateDbDir)
		defer os.RemoveAll(bp.stateDbDir)
	}

	// call post-prepare actions
	if err := actions.ExecuteAll("PostPrepare", bp); err != nil {
		return err
	}

	// create new iterator over substates and iterate
	log.Notice("Process blocks")
	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
	defer iter.Release()

	if err := bp.ProcessFirstBlock(iter); err != nil {
		return err
	}

	for iter.Next() {
		tx := iter.Value()
		// initiate first sync-period and block.
		// close off old block and possibly sync-periods
		if bp.block != tx.Block {
			if tx.Block > cfg.Last {
				break
			}
			bp.db.EndBlock()

			// switch to next sync-period if needed.
			// TODO: Revisit semantics - is this really necessary ????
			newSyncPeriod := tx.Block / cfg.SyncPeriodLength
			for bp.syncPeriod < newSyncPeriod {
				bp.db.EndSyncPeriod()
				bp.syncPeriod++
				bp.db.BeginSyncPeriod(bp.syncPeriod)
			}

			// Mark the beginning of a new block
			bp.block = tx.Block
			bp.db.BeginBlock(bp.block)
		}

		// check whether we have processed enough transaction
		// TODO: cfg.MaxNumTransactions should be a uint64 flag
		if cfg.MaxNumTransactions >= 0 && bp.totalTx >= uint64(cfg.MaxNumTransactions) {
			break
		}

		// process transaction
		if err := utils.ProcessTx(bp.db, cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
			log.Criticalf("\tFailed processing transaction: %v", err)
			return err
		}

		bp.totalGas.Add(bp.totalGas, new(big.Int).SetUint64(tx.Substate.Result.GasUsed))

		// call post-transaction actions
		if err := actions.ExecuteAll("PostTransaction", bp); err != nil {
			return err
		}
		bp.totalTx++
	}
	bp.db.EndBlock()
	bp.db.EndSyncPeriod()

	log.Noticef("%v errors found.", utils.NumErrors)

	// call post-processing actions
	if err := actions.ExecuteAll("PostProcessing", bp); err != nil {
		return err
	}

	// close the DB and print disk usage
	log.Info("Close StateDB")
	if err := bp.db.Close(); err != nil {
		return fmt.Errorf("Failed to close database: %v", err)
	}

	return err
}
