// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package statedb

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/executor/extension/validator"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeArchiveInquirer creates an extension running historic queries against
// archive states in the background to the main executor process.
func MakeArchiveInquirer(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeArchiveInquirer(cfg, logger.NewLogger(cfg.LogLevel, "Archive Inquirer"))
}

func makeArchiveInquirer(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	if cfg.ArchiveQueryRate <= 0 {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return &archiveInquirer{
		ArchiveDbTxProcessor: executor.MakeArchiveDbTxProcessor(cfg),
		cfg:                  cfg,
		log:                  log,
		throttler:            newThrottler(cfg.ArchiveQueryRate),
		finished:             utils.MakeEvent(),
		history:              newBuffer[historicTransaction](cfg.ArchiveMaxQueryAge),
		validator:            validator.MakeArchiveDbValidator(cfg, validator.ValidateTxTarget{WorldState: true, Receipt: true}),
	}
}

type archiveInquirer struct {
	extension.NilExtension[txcontext.TxContext]
	*executor.ArchiveDbTxProcessor

	cfg   *utils.Config
	log   logger.Logger
	state state.StateDB

	// Buffer for historic queries to sample from
	history      *circularBuffer[historicTransaction]
	historyMutex sync.Mutex

	// Worker control
	throttler *throttler
	finished  utils.Event
	done      sync.WaitGroup

	// Counters for throughput reporting
	transactionCounter         atomic.Uint64
	gasCounter                 atomic.Uint64
	totalQueryTimeMilliseconds atomic.Uint64

	validator executor.Extension[txcontext.TxContext]
}

func (i *archiveInquirer) PreRun(_ executor.State[txcontext.TxContext], ctx *executor.Context) error {
	if !i.cfg.ArchiveMode {
		i.finished.Signal()
		return fmt.Errorf("can not run archive queries without enabled archive (missing --%s flag)", utils.ArchiveModeFlag.Name)
	}
	i.state = ctx.State
	numWorkers := i.cfg.Workers
	if numWorkers < 1 {
		numWorkers = 1
	}
	i.done.Add(1 + numWorkers)
	for j := 0; j < numWorkers; j++ {
		go i.runInquiry(ctx.ErrorInput)
	}
	go i.runProgressReport()
	return nil
}

func (i *archiveInquirer) PostTransaction(state executor.State[txcontext.TxContext], _ *executor.Context) error {
	// We only sample the very first transaction in each block since other transactions
	// may depend on the effects of its predecessors in the same block.
	if state.Transaction != 0 {
		return nil
	}

	// Add current transaction as a candidate for replays.
	i.historyMutex.Lock()
	defer i.historyMutex.Unlock()
	i.history.Add(historicTransaction{
		block:  state.Block - 1,
		number: state.Transaction,
		data:   state.Data,
	})
	return nil
}

func (i *archiveInquirer) PostRun(executor.State[txcontext.TxContext], *executor.Context, error) error {
	i.finished.Signal()
	i.done.Wait()
	return nil
}

func (i *archiveInquirer) getRandomTransaction(rnd *rand.Rand) (historicTransaction, bool) {
	i.historyMutex.Lock()
	defer i.historyMutex.Unlock()
	size := i.history.Size()
	if size == 0 {
		return historicTransaction{}, false
	}
	return i.history.Get(int(rnd.Int31n(int32(size)))), true
}

func (i *archiveInquirer) runInquiry(errCh chan error) {
	defer i.done.Done()
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	for !i.finished.HasHappened() {
		if i.throttler.shouldRunNow() {
			i.doInquiry(rnd, errCh)
		} else {
			select {
			case <-time.After(10 * time.Millisecond):
				// nothing
			case <-i.finished.Wait():
				return
			}
		}
	}
}

func (i *archiveInquirer) doInquiry(rnd *rand.Rand, errCh chan error) {
	// Pick a random tx that is covered by the current archive block height.
	tx, found := i.getRandomTransaction(rnd)
	for found {
		height, empty, err := i.state.GetArchiveBlockHeight()
		if err != nil {
			i.log.Warningf("failed to obtain archive block height: %v", err)
			return
		}
		if empty {
			i.log.Warning("cannot run inquiry - archive is empty")
		}
		if !empty && uint64(tx.block) <= height {
			break
		}
		tx, found = i.getRandomTransaction(rnd)
	}
	if !found {
		return
	}

	// Perform historic query.
	archive, err := i.getArchive(uint64(tx.block), uint32(tx.number))
	if err != nil {
		// ArchiveInquirer should not end the app, hence we just send the error to the errorLogger
		errCh <- err
		return
	}

	defer func() {
		err = archive.EndTransaction()
		if err != nil {
			errCh <- fmt.Errorf("cannot end archive inquirer transaction; %w", err)
		}
		err = archive.Release()
		if err != nil {
			errCh <- fmt.Errorf("cannot release archive inside archive inquirer; %w", err)
		}

	}()

	state := executor.State[txcontext.TxContext]{
		Block:       tx.block,
		Transaction: tx.number,
		Data:        tx.data,
	}
	ctx := &executor.Context{
		Archive:    archive,
		ErrorInput: errCh,
	}

	// input validation
	err = i.validator.PreTransaction(state, ctx)
	if err != nil {
		// ArchiveInquirer should not end the app, hence we just send the error to the errorLogger
		errCh <- err
		return
	}

	start := time.Now()
	err = i.Process(state, ctx)
	if err != nil {
		// ArchiveInquirer should not end the app, hence we just send the error to the errorLogger
		errCh <- err
		return
	}
	duration := time.Since(start)

	// output validation
	err = i.validator.PostTransaction(state, ctx)
	if err != nil {
		// ArchiveInquirer should not end the app, hence we just send the error to the errorLogger
		errCh <- err
		return
	}

	i.transactionCounter.Add(1)
	i.gasCounter.Add(tx.data.GetResult().GetGasUsed())
	i.totalQueryTimeMilliseconds.Add(uint64(duration.Milliseconds()))
}

func (i *archiveInquirer) runProgressReport() {
	defer i.done.Done()
	lastTime := time.Now()
	lastTx := uint64(0)
	lastGas := uint64(0)
	lastDuration := uint64(0)

	start := time.Now()
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			curTx := i.transactionCounter.Load()
			curGas := i.gasCounter.Load()
			curDuration := i.totalQueryTimeMilliseconds.Load()

			delta := now.Sub(lastTime).Seconds()
			tps := float64(curTx-lastTx) / delta
			gps := float64(curGas-lastGas) / delta
			averageDuration := float64(curDuration-lastDuration) / float64(curTx-lastTx)
			i.log.Infof("Archive throughput: t=%ds, %.2f Tx/s, %.2f MGas/s, average duration %.2f ms",
				int(now.Sub(start).Round(time.Second).Seconds()), tps, gps/10e6, averageDuration,
			)

			lastTime = now
			lastTx = curTx
			lastGas = curGas
			lastDuration = curDuration
		case <-i.finished.Wait():
			return
		}
	}
}

func (i *archiveInquirer) getArchive(blk uint64, tx uint32) (state.NonCommittableStateDB, error) {
	archive, err := i.state.GetArchiveState(blk)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain access to archive blk %d, tx %d: %w", blk, tx, err)
	}

	err = archive.BeginTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("cannot begin transaction blk: %d, tx: %d; %w", blk, tx, err)
	}

	return archive, nil
}

type historicTransaction struct {
	block  int
	number int
	data   txcontext.TxContext
}

type circularBuffer[T any] struct {
	data []T
	head int
}

func newBuffer[T any](capacity int) *circularBuffer[T] {
	return &circularBuffer[T]{
		data: make([]T, 0, capacity),
	}
}

func (b *circularBuffer[T]) Size() int {
	return len(b.data)
}

func (b *circularBuffer[T]) Add(element T) {
	if cap(b.data) == 0 {
		return
	}
	if len(b.data) < cap(b.data) {
		b.data = append(b.data, element)
		return
	}
	b.data[b.head] = element
	b.head = (b.head + 1) % cap(b.data)
}

func (b *circularBuffer[T]) Get(pos int) T {
	return b.data[pos]
}

type throttler struct {
	transactionsPerSecond int
	lastUpdate            time.Time
	pending               float64
	mutex                 sync.Mutex
}

func newThrottler(rate int) *throttler {
	return &throttler{
		transactionsPerSecond: rate,
		lastUpdate:            time.Now(),
	}
}

func (t *throttler) shouldRunNow() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.pending < 1 {
		// Replenish pending transactions.
		now := time.Now()
		delta := now.Sub(t.lastUpdate)
		t.lastUpdate = now
		t.pending += float64(t.transactionsPerSecond) * delta.Seconds()
	}
	if t.pending >= 1 {
		t.pending -= 1
		return true
	}
	return false
}
