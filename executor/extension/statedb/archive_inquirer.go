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
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

// MakeArchiveInquirer creates an extension running historic queries against
// archive states in the background to the main executor process.
func MakeArchiveInquirer(cfg *utils.Config) executor.Extension[*substate.Substate] {
	return makeArchiveInquirer(cfg, logger.NewLogger(cfg.LogLevel, "Archive Inquirer"))
}

func makeArchiveInquirer(cfg *utils.Config, log logger.Logger) executor.Extension[*substate.Substate] {
	if cfg.ArchiveQueryRate <= 0 {
		return extension.NilExtension[*substate.Substate]{}
	}
	return &archiveInquirer{
		ArchiveDbProcessor: executor.MakeArchiveDbProcessor(cfg),
		cfg:                cfg,
		log:                log,
		throttler:          newThrottler(cfg.ArchiveQueryRate),
		finished:           utils.MakeEvent(),
		history:            newBuffer[historicTransaction](cfg.ArchiveMaxQueryAge),
		validator:          validator.MakeArchiveDbValidator(cfg),
	}
}

type archiveInquirer struct {
	extension.NilExtension[*substate.Substate]
	*executor.ArchiveDbProcessor

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

	validator executor.Extension[*substate.Substate]
}

func (i *archiveInquirer) PreRun(_ executor.State[*substate.Substate], ctx *executor.Context) error {
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

func (i *archiveInquirer) PostTransaction(state executor.State[*substate.Substate], _ *executor.Context) error {
	// We only sample the very first transaction in each block since other transactions
	// may depend on the effects of its predecessors in the same block.
	if state.Transaction != 0 {
		return nil
	}

	// Add current transaction as a candidate for replays.
	i.historyMutex.Lock()
	defer i.historyMutex.Unlock()
	i.history.Add(historicTransaction{
		block:    state.Block - 1,
		number:   state.Transaction,
		substate: state.Data,
	})
	return nil
}

func (i *archiveInquirer) PostRun(executor.State[*substate.Substate], *executor.Context, error) error {
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
	// Pick a random transaction that is covered by the current archive block height.
	transaction, found := i.getRandomTransaction(rnd)
	for found {
		height, empty, err := i.state.GetArchiveBlockHeight()
		if err != nil {
			i.log.Warningf("failed to obtain archive block height: %v", err)
			return
		}
		if !empty && uint64(transaction.block) <= height {
			break
		}
		transaction, found = i.getRandomTransaction(rnd)
	}
	if !found {
		return
	}

	// Perform historic query.
	archive, err := i.state.GetArchiveState(uint64(transaction.block))
	if err != nil {
		errCh <- fmt.Errorf("failed to obtain access to archive at block height %d: %v", transaction.block, err)
		return
	}
	defer archive.Release()

	state := executor.State[*substate.Substate]{
		Block:       transaction.block,
		Transaction: transaction.number,
		Data:        transaction.substate,
	}
	ctx := &executor.Context{
		Archive:    archive,
		ErrorInput: errCh,
	}

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
	i.gasCounter.Add(transaction.substate.Result.GasUsed)
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

type historicTransaction struct {
	block    int
	number   int
	substate *substate.Substate
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
