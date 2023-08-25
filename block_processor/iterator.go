package blockprocessor

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
)

// how often will we print progress
const progressPeriod = 15 * time.Second

// BasicIterator iterates over substates and saves info about them
func BasicIterator(iter substate.SubstateIterator, actions ExtensionList, bp *BlockProcessor) error {
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
		if _, err = utils.ProcessTx(bp.db, bp.cfg, bp.tx.Block, bp.tx.Transaction, bp.tx.Substate); err != nil {
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

type VmAdbIterator struct {
	cfg                  *utils.Config
	log                  *logging.Logger
	iter                 substate.SubstateIterator
	actions              ExtensionList
	lastBlock            uint64
	db                   state.StateDB
	unitedTransactionsCh chan []*substate.Transaction
	totalGas             *big.Int
	totalTxs             *big.Int
	totalGasCh           chan uint64
	totalTxCh            chan int
	closeCh              chan any
	errCh                chan error

	wg *sync.WaitGroup
}

func newVmAdbIterator(iter substate.SubstateIterator, actions ExtensionList, bp *BlockProcessor) *VmAdbIterator {
	return &VmAdbIterator{
		cfg:                  bp.cfg,
		log:                  logger.NewLogger(bp.cfg.LogLevel, "VmAdb-Iterator"),
		iter:                 iter,
		actions:              actions,
		lastBlock:            bp.cfg.Last,
		db:                   bp.db,
		unitedTransactionsCh: make(chan []*substate.Transaction, 10*bp.cfg.Workers),
		totalGas:             new(big.Int),
		totalTxs:             new(big.Int),
		totalGasCh:           make(chan uint64, 10*bp.cfg.Workers),
		totalTxCh:            make(chan int, 10*bp.cfg.Workers),
		closeCh:              make(chan any, 1),
		errCh:                make(chan error, 1),
		wg:                   new(sync.WaitGroup),
	}
}

// VmAdbIterate is an iterator for vm-adb tool
func VmAdbIterate(iter substate.SubstateIterator, actions ExtensionList, bp *BlockProcessor) error {
	it := newVmAdbIterator(iter, actions, bp)
	go it.countProgress()
	go it.groupTransactions()

	defer it.passValuesToBp(bp)

	for i := 0; i < it.cfg.Workers; i++ {
		it.wg.Add(1)
		go it.runBlocks()
	}

	select {
	case err := <-it.errCh:
		it.Close()
		it.wg.Wait()
		return err
	default:
	}

	it.wg.Wait()

	return nil
}

// Close sends the exit signal
func (it *VmAdbIterator) Close() {
	select {
	case <-it.closeCh:
		return
	default:
		close(it.closeCh)

	}
}

// countProgress is a thread for counting total gas and number of transactions
func (it *VmAdbIterator) countProgress() {
	var (
		txCount int
		gas     uint64
		// for thread safety we need fresh instance of logger
		log             = logger.NewLogger(it.cfg.LogLevel, "vm-adb-progress")
		processedBlocks = it.cfg.First
		start           = time.Now()
		ticker          = time.NewTicker(progressPeriod)
	)

	it.wg.Add(1)
	defer it.wg.Done()

	for {
		select {
		case <-it.closeCh:
			return
		case <-ticker.C:
			it.printProgress(log, it.totalTxs.Uint64(), it.totalGas.Uint64(), processedBlocks, start)
		case txCount = <-it.totalTxCh:
			it.totalTxs.SetUint64(it.totalTxs.Uint64() + uint64(txCount))

			// check whether we have processed enough transaction
			// TODO: cfg.MaxNumTransactions should be a uint64 flag
			if it.cfg.MaxNumTransactions >= 0 && it.totalTxs.Uint64() >= uint64(it.cfg.MaxNumTransactions) {
				break
			}

			processedBlocks++

		case gas = <-it.totalGasCh:
			it.totalGas.SetUint64(it.totalGas.Uint64() + gas)
		}
	}
}

// printProgress depending on how often we set the print using progressPeriod
func (it *VmAdbIterator) printProgress(log *logging.Logger, txs uint64, gas uint64, block uint64, start time.Time) {
	s := time.Since(start).Seconds()
	txRate := float64(txs) / s

	log.Infof("Elapsed time: %v, at block %v, used gas: %v, speed: (~ %.1f Tx/s), tx processed: %v", time.Since(start).Round(1*time.Second), block, gas, txRate, txs)

}

// groupTransactions by block number - everytime a tx with new block number occurs, the current txPool is sent to process
func (it *VmAdbIterator) groupTransactions() {
	var (
		currentBlock = it.cfg.First
		tx           *substate.Transaction
		txPool       []*substate.Transaction
	)

	it.wg.Add(1)

	defer func() {
		close(it.unitedTransactionsCh)
		it.wg.Done()
	}()

	for it.iter.Next() {
		select {
		case <-it.closeCh:
			return
		default:
		}

		tx = it.iter.Value()

		// if we have gotten all txs from block, we send them to processes
		if tx.Block != currentBlock {

			currentBlock = tx.Block
			select {
			case it.unitedTransactionsCh <- txPool:
				// was this last block?
				if tx.Block > it.lastBlock {
					return
				}
			default:
			}
			// reset the buffer
			txPool = []*substate.Transaction{}
		}

		txPool = append(txPool, tx)
	}

}

// runBlocks reads the united transactions and sends them to processing
func (it *VmAdbIterator) runBlocks() {
	var (
		transactions []*substate.Transaction
		err          error
	)

	defer it.wg.Done()

	for {
		select {
		case <-it.closeCh:
			return
		case transactions = <-it.unitedTransactionsCh:
			// chanel has been closed and its empty
			if transactions == nil {
				// stop only when all transactions have been processed
				it.Close()
				return
			}
			if err = it.processTransactions(transactions); err != nil {
				it.errCh <- err
				return
			}

		}
	}
}

// processTransactions united by block number and send info about them to respective channels
func (it *VmAdbIterator) processTransactions(transactions []*substate.Transaction) error {
	block := transactions[0].Block - 1
	archive, err := it.db.GetArchiveState(block)
	if err != nil {
		return err
	}

	archive.BeginBlock(block)

	for _, tx := range transactions {
		// process transaction
		if _, err = utils.ProcessTx(archive, it.cfg, block, tx.Transaction, tx.Substate); err != nil {
			return fmt.Errorf("failed processing transaction; %v", err)
		}

		it.totalGasCh <- tx.Substate.Result.GasUsed

	}

	it.totalTxCh <- len(transactions)

	if err = archive.Close(); err != nil {
		it.errCh <- fmt.Errorf("cannot close archive number %v; %v", block, err)
	}

	return nil
}

// passValuesToBp for end progress report
func (it *VmAdbIterator) passValuesToBp(bp *BlockProcessor) {
	bp.totalGas = it.totalGas
	bp.totalTx = it.totalTxs.Uint64()
}
