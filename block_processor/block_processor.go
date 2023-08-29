package blockprocessor

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
)

type BlockProcessor struct {
	Cfg        *utils.Config   // configuration
	Log        *logging.Logger // logger
	stateDbDir string          // directory of the StateDB
	Db         state.StateDB   // StateDB
	TotalTx    *big.Int        // total number of transactions so far
	TotalGas   *big.Int        // total gas consumed so far
	Block      uint64
	actions    ExtensionList
}

// NewBlockProcessor creates a new block processor instance
func NewBlockProcessor(cfg *utils.Config, actions ExtensionList, name string) *BlockProcessor {

	return &BlockProcessor{
		Cfg:      cfg,
		Log:      logger.NewLogger(cfg.LogLevel, name),
		TotalGas: new(big.Int),
		TotalTx:  new(big.Int),
		actions:  actions,
	}
}

// Prepare opens substateDb and primes World-State
func (bp *BlockProcessor) Prepare() error {
	var err error

	// open substate database
	bp.Log.Notice("Open substate database")
	substate.SetSubstateDb(bp.Cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()

	bp.Log.Notice("Open StateDb")
	bp.Db, bp.stateDbDir, err = utils.PrepareStateDB(bp.Cfg)
	if err != nil {
		return err
	}

	if !bp.Cfg.SkipPriming && bp.Cfg.StateDbSrc == "" {
		if err = utils.LoadWorldStateAndPrime(bp.Db, bp.Cfg, bp.Cfg.First-1); err != nil {
			return fmt.Errorf("priming failed. %v", err)
		}
	}

	// call post-prepare actions
	if err = bp.ExecuteExtension("PostPrepare"); err != nil {
		return fmt.Errorf("cannot execute 'post-prepare' extensions")
	}

	return nil
}

// Config provides the processes configuration parsed by this block processor
// from command line parameters, default values, and other sources.
func (bp *BlockProcessor) Config() *utils.Config {
	return bp.Cfg
}

func (bp *BlockProcessor) ExecuteExtension(method string) error {
	return bp.actions.executeExtensions(method, bp)
}

// Exit is always executed in defer
func (bp *BlockProcessor) Exit() error {
	substate.CloseSubstateDB()

	if err := bp.ExecuteExtension("Exit"); err != nil {
		return fmt.Errorf("cannot execute 'exit' extensions; %v", err)
	}

	utils.PrintEvmStatistics(bp.Cfg)

	return nil
}
