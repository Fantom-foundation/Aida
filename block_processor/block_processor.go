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
	log        *logging.Logger // logger
	stateDbDir string          // directory of the StateDB
	db         state.StateDB   // StateDB
	totalTx    *big.Int        // total number of transactions so far
	totalGas   *big.Int        // total gas consumed so far
	block      uint64
	actions    ExtensionList
}

// NewBlockProcessor creates a new block processor instance
func NewBlockProcessor(cfg *utils.Config, actions ExtensionList, name string) *BlockProcessor {

	return &BlockProcessor{
		Cfg:      cfg,
		log:      logger.NewLogger(cfg.LogLevel, name),
		totalGas: new(big.Int),
		totalTx:  new(big.Int),
		actions:  actions,
	}
}

// Prepare opens substateDb and primes World-State
func (bp *BlockProcessor) Prepare() error {
	var err error

	// open substate database
	bp.log.Notice("Open substate database")
	substate.SetSubstateDb(bp.Cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()

	bp.log.Notice("Open StateDb")
	bp.db, bp.stateDbDir, err = utils.PrepareStateDB(bp.Cfg)
	if err != nil {
		return err
	}

	if !bp.Cfg.SkipPriming && bp.Cfg.StateDbSrc == "" {
		if err = utils.LoadWorldStateAndPrime(bp.db, bp.Cfg, bp.Cfg.First-1); err != nil {
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

func (bp *BlockProcessor) Db() state.StateDB {
	return bp.db
}

func (bp *BlockProcessor) AddTotalGas(delta uint64) {
	bp.totalGas.SetUint64(bp.totalGas.Uint64() + delta)
}

func (bp *BlockProcessor) AddTotalTx(delta uint64) {
	bp.totalTx.SetUint64(bp.totalTx.Uint64() + delta)
}

func (bp *BlockProcessor) TotalTx() uint64 {
	return bp.totalTx.Uint64()
}

func (bp *BlockProcessor) Log() *logging.Logger {
	return bp.log
}

func (bp *BlockProcessor) ExecuteExtension(method string) error {
	return bp.actions.executeExtensions(method, bp)
}

func (bp *BlockProcessor) Block() uint64 {
	return bp.block
}

func (bp *BlockProcessor) SetBlock(block uint64) {
	bp.block = block
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
