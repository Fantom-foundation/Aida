package vm_adb

import (
	"fmt"
	"sync"

	blockprocessor "github.com/Fantom-foundation/Aida/block_processor"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RunVmAdb performs block processing on an ArchiveDb
func RunVmAdb(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	actions := blockprocessor.NewExtensionList([]blockprocessor.ProcessorExtensions{
		blockprocessor.NewProgressReportExtension(cfg),
		blockprocessor.NewValidationExtension(),
		blockprocessor.NewProfileExtension(),
	})

	bp := NewVmAdb(cfg, actions)
	return bp.Run()
}

type VmAdb struct {
	*blockprocessor.BlockProcessor
	unitedTransactionsCh chan []*substate.Transaction
	totalGasCh           chan uint64
	totalTxCh            chan int
	closeCh              chan any
	errCh                chan error
	wg                   *sync.WaitGroup
}

func NewVmAdb(cfg *utils.Config, actions blockprocessor.ExtensionList) *VmAdb {
	return &VmAdb{
		BlockProcessor:       blockprocessor.NewBlockProcessor(cfg, actions, "Aida VM ADb"),
		unitedTransactionsCh: make(chan []*substate.Transaction, 10*cfg.Workers),
		totalGasCh:           make(chan uint64, 10*cfg.Workers),
		totalTxCh:            make(chan int, 10*cfg.Workers),
		closeCh:              make(chan any, 1),
		errCh:                make(chan error, 1),
		wg:                   new(sync.WaitGroup),
	}
}

func (bp *VmAdb) Run() error {
	var err error

	// call init actions
	if err = bp.ExecuteExtension("Init"); err != nil {
		return err
	}

	// close actions when return
	defer func() error {
		close(bp.errCh)
		close(bp.totalGasCh)
		close(bp.totalTxCh)
		return bp.Exit()
	}()

	// prepare statedb and priming
	if err = bp.Prepare(); err != nil {
		return fmt.Errorf("cannot prepare block processor; %v", err)
	}

	bp.Log.Notice("Process blocks")
	iter := substate.NewSubstateIterator(bp.Cfg.First, bp.Cfg.Workers)
	defer iter.Release()

	// start threads
	go bp.Iterate(iter, bp.Cfg.First, bp.Cfg.Last)
	go bp.countProgress(uint64(bp.Cfg.MaxNumTransactions), bp.Cfg.First)

	// start workers
	for i := 0; i < bp.Config().Workers; i++ {
		bp.wg.Add(1)
		// for thread safety, we need to copy value of config to each worker
		go bp.runBlocks(*bp.Config())
	}

	select {
	case err = <-bp.errCh:
		bp.Close()
	default:
	}

	bp.wg.Wait()

	return err
}

// Iterate over substates, unite transactions it by block number and then send it to process
func (bp *VmAdb) Iterate(iter substate.SubstateIterator, firstBlock, lastBlock uint64) {
	var (
		currentBlock = firstBlock
		tx           *substate.Transaction
		txPool       []*substate.Transaction
	)

	bp.wg.Add(1)

	defer func() {
		close(bp.unitedTransactionsCh)
		bp.wg.Done()
	}()

	for iter.Next() {
		select {
		case <-bp.closeCh:
			return
		default:
		}

		tx = iter.Value()

		// if we have gotten all txs from block, we send them to processes
		if tx.Block != currentBlock {
			currentBlock = tx.Block
			select {
			case bp.unitedTransactionsCh <- txPool:
				// was this last block?
				if tx.Block > lastBlock {
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

// countProgress is a thread for counting total gas and number of transactions
// this is the only thread we access these two variables,
// hence here we call the PostBlock extensions for ProgressReportExtension
func (bp *VmAdb) countProgress(maximumTxs uint64, firstBlock uint64) {
	var (
		txCount int
		gas     uint64
		err     error
	)

	bp.Block = firstBlock

	bp.wg.Add(1)
	defer bp.wg.Done()

	for {
		select {
		case <-bp.closeCh:
			return
		case txCount = <-bp.totalTxCh:
			bp.TotalTx.SetUint64(bp.TotalTx.Uint64() + uint64(txCount))

			// check whether we have processed enough transaction
			// TODO: cfg.MaxNumTransactions should be a uint64 flag
			if maximumTxs >= 0 && bp.TotalTx.Uint64() >= maximumTxs {
				break
			}

			bp.Block++

			if err = bp.ExecuteExtension("PostBlock"); err != nil {
				bp.errCh <- fmt.Errorf("cannot execute 'post-block' extension; %v", err)
				return
			}

		case gas = <-bp.totalGasCh:
			bp.TotalGas.SetUint64(bp.TotalGas.Uint64() + gas)
		}
	}
}

// runBlocks reads the united transactions and sends them to processing
func (bp *VmAdb) runBlocks(cfg utils.Config) {
	var (
		transactions []*substate.Transaction
		err          error
	)

	defer bp.wg.Done()

	for {
		select {
		case <-bp.closeCh:
			return
		case transactions = <-bp.unitedTransactionsCh:
			// chanel has been closed and its empty
			if transactions == nil {
				// stop only when all transactions have been processed
				bp.Close()
				return
			}
			if err = bp.processTransactions(transactions, &cfg); err != nil {
				bp.errCh <- err
				return
			}

		}
	}
}

// processTransactions united by block number and send info about them to respective channels
func (bp *VmAdb) processTransactions(transactions []*substate.Transaction, cfg *utils.Config) error {
	block := transactions[0].Block - 1
	archive, err := bp.Db.GetArchiveState(block)
	if err != nil {
		return err
	}

	archive.BeginBlock(block)

	for _, tx := range transactions {
		// process transaction
		if _, err = utils.ProcessTx(archive, cfg, block, tx.Transaction, tx.Substate); err != nil {
			return fmt.Errorf("failed processing transaction; %v", err)
		}

		bp.totalGasCh <- tx.Substate.Result.GasUsed

	}

	bp.totalTxCh <- len(transactions)

	if err = archive.Close(); err != nil {
		bp.errCh <- fmt.Errorf("cannot close archive number %v; %v", block, err)
	}

	return nil
}

// Close sends the exit signal
func (bp *VmAdb) Close() {
	select {
	case <-bp.closeCh:
		return
	default:
		close(bp.closeCh)

	}
}
