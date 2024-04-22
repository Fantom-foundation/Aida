// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utildb

import (
	"errors"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/state/proxy"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const channelSize = 100000 // size of deletion channel

type txLivelinessResult struct {
	liveliness []proxy.ContractLiveliness
	tx         *substate.Transaction
}

// readAccounts reads contracts which were suicided or created and adds them to lists
func readAccounts(cllArr []proxy.ContractLiveliness, deleteHistory *map[common.Address]bool) ([]common.Address, []common.Address) {
	des := make(map[common.Address]bool)
	res := make(map[common.Address]bool)

	for _, contract := range cllArr {
		addr := contract.Addr
		if contract.IsDeleted {
			// if a contract was resurrected before suicided in the same tx,
			// only keep the last action.
			if _, found := res[addr]; found {
				delete(res, addr)
			}
			(*deleteHistory)[addr] = true // meta list
			des[addr] = true
		} else {
			// if a contract was suicided before resurrected in the same tx,
			// only keep the last action.
			if _, found := des[addr]; found {
				delete(des, addr)
			}
			// an account is considered as resurrected if it was recently deleted.
			if recentlyDeleted, found := (*deleteHistory)[addr]; found && recentlyDeleted {
				(*deleteHistory)[addr] = false
				res[addr] = true
			} else if found && !recentlyDeleted {
			}
		}
	}

	var deletedAccounts []common.Address
	var resurrectedAccounts []common.Address

	for addr := range des {
		deletedAccounts = append(deletedAccounts, addr)
	}
	for addr := range res {
		resurrectedAccounts = append(resurrectedAccounts, addr)
	}
	return deletedAccounts, resurrectedAccounts
}

// genDeletedAccountsTask process a transaction substate then records self-destructed accounts
// and resurrected accounts to a database.
func genDeletedAccountsTask(tx *substate.Transaction, processor executor.TxProcessor, cfg *utils.Config) ([]proxy.ContractLiveliness, error) {
	ch := make(chan proxy.ContractLiveliness, channelSize)
	var statedb state.StateDB
	var err error
	ss := substatecontext.NewTxContext(tx.Substate)

	statedb, err = state.MakeOffTheChainStateDB(ss.GetInputState(), tx.Block, state.NewChainConduit(cfg.ChainID == utils.EthereumChainID, utils.GetChainConfig(cfg.ChainID)))
	if err != nil {
		return nil, err
	}

	//wrapper
	statedb = proxy.NewDeletionProxy(statedb, ch, cfg.LogLevel)

	_, err = processor.ProcessTransaction(statedb, int(tx.Block), tx.Transaction, ss)
	if err != nil {
		return nil, nil
	}

	close(ch)

	livelinessArr := make([]proxy.ContractLiveliness, 0)
	for liveliness := range ch {
		livelinessArr = append(livelinessArr, liveliness)
	}

	return livelinessArr, nil
}

// GenDeletedAccountsAction replays transactions and record self-destructed accounts and resurrected accounts.
// Uses round-robin task assignment system to workers to keep order while utilizing parallelism.
func GenDeletedAccountsAction(cfg *utils.Config, ddb *substate.DestroyedAccountDB, firstBlock uint64, lastBlock uint64) error {
	err := utils.StartCPUProfile(cfg)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "Generate Deleted Accounts")
	log.Noticef("Generate deleted accounts from block %v to block %v", firstBlock, lastBlock)

	processor := executor.MakeTxProcessor(cfg)

	wg := sync.WaitGroup{}
	stopChan := make(chan struct{})
	errChan := make(chan error)

	iter := substate.NewSubstateIterator(firstBlock, cfg.Workers)
	defer iter.Release()

	//error handling routine
	encounteredErrors := errorHandler(stopChan, errChan)

	// feeder to send tasks to workers
	workerInputChannels := taskFeeder(&wg, cfg, iter, lastBlock, stopChan, log)

	// prepare workers to process transactions
	workerOutputChannels := launchWorkers(&wg, cfg, workerInputChannels, processor, stopChan, errChan)

	// collect results from workers and orders them
	orderedResults := resultCollector(&wg, cfg, workerOutputChannels, stopChan)

	// process ordered txLivelinessResults
	resolveDeletionsAndResurrections(ddb, orderedResults, stopChan, errChan)

	// wait until feeder, workers and collector are done
	wg.Wait()

	// notify error handler to stop
	close(errChan)

	utils.StopCPUProfile(cfg)

	// retrieve encounteredErrors from error handler
	err = <-encounteredErrors
	return err
}

// resolveDeletionsAndResurrections reads txLivelinessResults and resolves deletions and resurrections.
func resolveDeletionsAndResurrections(ddb *substate.DestroyedAccountDB, orderedResults chan txLivelinessResult, stopChan chan struct{}, errChan chan error) {
	var deleteHistory = make(map[common.Address]bool)
	defer func() {
		// explicitly set to nil to release memory as soon as possible
		deleteHistory = nil
	}()

	for {
		select {
		case <-stopChan:
			return
		case contract, ok := <-orderedResults:
			{
				if !ok {
					return
				}
				des, res := readAccounts(contract.liveliness, &deleteHistory)
				if len(des)+len(res) > 0 {
					err := ddb.SetDestroyedAccounts(contract.tx.Block, contract.tx.Transaction, des, res)
					if err != nil {
						errChan <- err
						return
					}
				}
			}
		}
	}
}

// resultCollector collects results from workers in round-robin fashion and sends them to a single channel.
func resultCollector(wg *sync.WaitGroup, cfg *utils.Config, workerOutputChannels map[int]chan txLivelinessResult, stopChan chan struct{}) chan txLivelinessResult {
	orderedResults := make(chan txLivelinessResult, cfg.Workers)
	wg.Add(1)
	go func() {
		defer close(orderedResults)
		defer wg.Done()

		// round-robin to collect results from workers
		for {
			for i := 0; i < cfg.Workers; i++ {
				select {
				case <-stopChan:
					return
				case res, ok := <-workerOutputChannels[i]:
					if !ok {
						return
					}

					// filter out txs with no liveliness actions
					if res.liveliness != nil && len(res.liveliness) > 0 {
						orderedResults <- res
					}
				}
			}
		}
	}()
	return orderedResults
}

// launchWorkers lauches workers to process transactions in parallel.
func launchWorkers(wg *sync.WaitGroup, cfg *utils.Config, workerInputChannels map[int]chan *substate.Transaction, processor executor.TxProcessor, stopChan chan struct{}, errChan chan error) map[int]chan txLivelinessResult {
	// channel for each worker to send results
	workerOutputChannels := make(map[int]chan txLivelinessResult)
	for i := 0; i < cfg.Workers; i++ {
		workerOutputChannels[i] = make(chan txLivelinessResult)
	}

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer func() {
				close(workerOutputChannels[workerId])
				wg.Done()
			}()

			for {
				select {
				case <-stopChan:
					return
				case tx, ok := <-workerInputChannels[workerId]:
					if !ok {
						return
					}
					// Process sorted transactions
					livelinessArr, err := genDeletedAccountsTask(tx, processor, cfg)
					if err != nil {
						errChan <- err
						return
					}

					select {
					case <-stopChan:
						return
					case workerOutputChannels[workerId] <- txLivelinessResult{livelinessArr, tx}:
					}
				}
			}
		}(i)
	}

	return workerOutputChannels
}

// taskFeeder feeds tasks to workers in round-robin fashion.
func taskFeeder(wg *sync.WaitGroup, cfg *utils.Config, iter substate.SubstateIterator, lastBlock uint64, stopChan chan struct{}, log logger.Logger) map[int]chan *substate.Transaction {
	wg.Add(1)

	// channel for each worker to get tasks for processing
	workerInputChannels := make(map[int]chan *substate.Transaction)
	for i := 0; i < cfg.Workers; i++ {
		workerInputChannels[i] = make(chan *substate.Transaction)
	}

	go func() {
		start := time.Now()
		sec := time.Since(start).Seconds()
		lastSec := time.Since(start).Seconds()
		txCount := uint64(0)
		lastTxCount := uint64(0)

		defer func() {
			wg.Done()
			// close inputs for workers
			for _, inputChan := range workerInputChannels {
				close(inputChan)
			}
		}()

		// Round-robin worker index
		nextWorkerIndex := 0
		for iter.Next() {
			select {
			case <-stopChan:
				return
			default:
			}

			tx := iter.Value()

			sec = time.Since(start).Seconds()
			diff := sec - lastSec

			if tx.Block > lastBlock {
				log.Infof("gen-del-acc: Total elapsed time: %.0f s, (Total ~%.1f Tx/s)", sec, float64(txCount)/sec)
				break
			}

			txCount++
			if diff >= 30 {
				numTx := txCount - lastTxCount
				lastTxCount = txCount
				log.Infof("gen-del-acc: Elapsed time: %.0f s, at block %v (~%.1f Tx/s)", sec, tx.Block, float64(numTx)/diff)
				lastSec = sec
			}

			if tx.Transaction < utils.PseudoTx && tx.Substate.Result.Status == types.ReceiptStatusSuccessful {
				// if not pseodo tx and completed successfully, send task to next worker in round-robin
				select {
				case <-stopChan:
					return
				case workerInputChannels[nextWorkerIndex] <- tx:
					nextWorkerIndex = (nextWorkerIndex + 1) % cfg.Workers
				}
			}
		}
	}()

	return workerInputChannels
}

// errorHandler collects errors from workers and returns them as a single error
// while closing the stopChan to signal other routines to stop.
func errorHandler(stopChan chan struct{}, errChan chan error) chan error {
	encounteredErrors := make(chan error)
	go func() {
		defer close(encounteredErrors)

		var result error
		firstErr := true

		defer func() {
			if firstErr {
				close(stopChan)
				firstErr = false
			}
		}()

		for {
			err, ok := <-errChan
			if !ok {
				encounteredErrors <- result
				return
			}
			if firstErr {
				close(stopChan)
				firstErr = false
			}

			result = errors.Join(result, err)
		}
	}()
	return encounteredErrors
}
