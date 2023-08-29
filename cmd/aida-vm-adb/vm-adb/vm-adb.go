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

	extensions := blockprocessor.NewExtensionList([]blockprocessor.ProcessorExtensions{
		blockprocessor.NewProgressReportExtension(),
		blockprocessor.NewValidationExtension(),
		blockprocessor.NewProfileExtension(),
	})

	bp := NewVmAdb(cfg, extensions)
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

func (adb *VmAdb) Run() error {
	// call init actions
	if err := adb.ExecuteExtension("Init"); err != nil {
		return err
	}

	// close actions when return
	defer func() error {
		close(adb.errCh)
		close(adb.totalGasCh)
		close(adb.totalTxCh)
		return adb.Exit()
	}()

	// prepare statedb and priming
	if err := adb.Prepare(); err != nil {
		return fmt.Errorf("cannot prepare block processor; %v", err)
	}

	adb.Log.Notice("Process blocks")
	iter := substate.NewSubstateIterator(adb.Cfg.First, adb.Cfg.Workers)
	defer iter.Release()

	// start threads
	go adb.Iterate(iter, adb.Cfg.First)
	go adb.checkProgress(uint64(adb.Cfg.MaxNumTransactions), adb.Cfg.First, adb.Cfg.Last)

	adb.Log.Infof("Starting %v workers", adb.Cfg.Workers)
	// start workers
	for i := 0; i < adb.Cfg.Workers; i++ {
		adb.wg.Add(1)
		// for thread safety, we need to copy value of config to each worker
		go adb.runBlocks(*adb.Cfg)
	}

	adb.wg.Wait()

	return nil
}

// Iterate over substates, unite transactions by block number and then send it to process
func (adb *VmAdb) Iterate(iter substate.SubstateIterator, firstBlock uint64) {
	var (
		currentBlock = firstBlock
		tx           *substate.Transaction
		txPool       []*substate.Transaction
	)

	adb.wg.Add(1)

	defer func() {
		adb.wg.Done()
	}()

	for iter.Next() {
		tx = iter.Value()

		// if we have gotten all txs from block, we send them to processes
		if tx.Block != currentBlock {
			currentBlock = tx.Block
			select {
			case <-adb.closeCh:
				return
			case adb.unitedTransactionsCh <- txPool:
			}
			// reset the buffer
			txPool = []*substate.Transaction{}
		}

		txPool = append(txPool, tx)
	}
}

// checkProgress is a thread for counting total gas, total number of transactions and completed blocks
// this is the only thread we access these two variables,
// hence here we call the PostBlock extensions for ProgressReportExtension.
// This thread sends signal to close the program once we complete enough blocks/txs
func (adb *VmAdb) checkProgress(maximumTxs uint64, firstBlock uint64, lastBlock uint64) {
	var (
		txCount int
		gas     uint64
		err     error
	)

	bp.Block = firstBlock

	adb.Block = firstBlock

	adb.wg.Add(1)
	defer func() {
		if err = adb.ExecuteExtension("PostProcessing"); err != nil {
			select {
			case <-adb.closeCh:
				return
			case adb.errCh <- fmt.Errorf("cannot execute 'post-processing' extension; %v", err):
				return
			}
		}
		adb.wg.Done()
		adb.Close()
	}()

	for {
		select {
		case <-adb.closeCh:
			return
		case txCount = <-adb.totalTxCh:
			adb.TotalTx.SetUint64(adb.TotalTx.Uint64() + uint64(txCount))

			// check whether we have processed enough transaction
			// TODO: cfg.MaxNumTransactions should be a uint64 flag
			if maximumTxs >= 0 && adb.TotalTx.Uint64() >= maximumTxs {
				return
			}

			adb.Block++

			// check whether we have processed enough blocks
			if adb.Block > lastBlock {
				return
			}

			if err = adb.ExecuteExtension("PostBlock"); err != nil {
				select {
				case <-adb.closeCh:
					return
				case adb.errCh <- fmt.Errorf("cannot execute 'post-block' extension; %v", err):
					return
				}
			}
		case gas = <-adb.totalGasCh:
			adb.TotalGas.SetUint64(adb.TotalGas.Uint64() + gas)
		}
	}
}

// runBlocks reads the united transactions and sends them to processing
func (adb *VmAdb) runBlocks(cfg utils.Config) {
	var (
		transactions []*substate.Transaction
		err          error
	)

	defer func() {
		adb.wg.Done()
	}()

	for {
		select {
		case <-adb.closeCh:
			return
		case transactions = <-adb.unitedTransactionsCh:
			if err = adb.processTransactions(transactions, &cfg); err != nil {
				select {
				case <-adb.closeCh:
					return
				case adb.errCh <- err:
					return
				}
			}

		}
	}
}

// processTransactions united by block number and send info about them to respective channels
func (adb *VmAdb) processTransactions(transactions []*substate.Transaction, cfg *utils.Config) error {
	block := transactions[0].Block - 1
	archive, err := adb.Db.GetArchiveState(block)
	if err != nil {
		return err
	}

	archive.BeginBlock(block)

	for _, tx := range transactions {
		// process transaction
		if _, err = utils.ProcessTx(archive, cfg, block, tx.Transaction, tx.Substate); err != nil {
			return fmt.Errorf("failed processing transaction; %v", err)
		}
		select {
		case <-adb.closeCh:
			return nil
		case adb.totalGasCh <- tx.Substate.Result.GasUsed:
		}

	}

	select {
	case <-adb.closeCh:
		return nil
	case adb.totalTxCh <- len(transactions):
	}

	if err = archive.Close(); err != nil {
		return fmt.Errorf("cannot close archive number %v; %v", block, err)
	}

	return nil
}

// Close sends the exit signal
func (adb *VmAdb) Close() {
	select {
	case <-adb.closeCh:
		return
	default:
		close(adb.closeCh)

	}
}
