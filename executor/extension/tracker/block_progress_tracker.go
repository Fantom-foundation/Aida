package tracker

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

const substateProgressTrackerReportFormat = "Track: block %d, memory %d, disk %d, interval_blk_rate %.2f, interval_tx_rate %.2f, interval_gas_rate %.2f, overall_blk_rate %.2f, overall_tx_rate %.2f, overall_gas_rate %.2f"

// MakeBlockProgressTracker creates a blockProgressTracker that depends on the
// PostBlock event and is only useful as part of a sequential evaluation.
func MakeBlockProgressTracker(cfg *utils.Config, reportFrequency int) executor.Extension[txcontext.TxContext] {
	if !cfg.TrackProgress {
		return extension.NilExtension[txcontext.TxContext]{}
	}

	if reportFrequency == 0 {
		reportFrequency = ProgressTrackerDefaultReportFrequency
	}

	return makeBlockProgressTracker(cfg, reportFrequency, logger.NewLogger(cfg.LogLevel, "ProgressTracker"))
}

func makeBlockProgressTracker(cfg *utils.Config, reportFrequency int, log logger.Logger) *blockProgressTracker {
	var pub *RedisPublisher
	if cfg.RegisterRun != "" {
		pub, _ = MakeRedisPublisher("95.217.204.177", 6379, "aida")
	}

	return &blockProgressTracker{
		progressTracker:   newProgressTracker[txcontext.TxContext](cfg, reportFrequency, log),
		lastReportedBlock: int(cfg.First) - (int(cfg.First) % reportFrequency),
		pub: pub,
	}
}

// blockProgressTracker logs progress every XXX blocks depending on reportFrequency.
// Default is 100_000 blocks. This is mainly used for gathering information about process.
type blockProgressTracker struct {
	*progressTracker[txcontext.TxContext]
	overallInfo       substateProcessInfo
	lastIntervalInfo  substateProcessInfo
	lastReportedBlock int

	pub	*RedisPublisher
}

type substateProcessInfo struct {
	numTransactions uint64
	gas             uint64
}

type Message map[string]any
type Publisher interface {
	Publish(Message) error
}

type NilPublisher struct {}
func (_ *NilPublisher) Publish(m Message) error {
	return nil
}

type RedisPublisher struct {
	NilPublisher
	r *redis.Client
	topic string
	log logger.Logger
}

func MakeRedisPublisher(addr string, port int, topic string) (*RedisPublisher, error) {
	log := logger.NewLogger("Error", "RedisPublisher")

	r := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", addr, port),
	})

	_, err := r.Ping().Result()
	if err != nil {
		log.Fatalf("Unable to connect to Redis, %s", err)
		return nil, err
	}

	log.Noticef("Connected to Redis server, %s:%d", addr, port)

	return &RedisPublisher{r: r, topic: topic, log: log}, nil
}

func (pub *RedisPublisher) Publish(m Message) error {
	err := pub.r.XAdd(&redis.XAddArgs{
		Stream: pub.topic,
		Values: m,
	}).Err()

	return err
}

// PostTransaction increments number of transactions and saves gas used in last substate.
func (t *blockProgressTracker) PostTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.overallInfo.numTransactions++
	if ctx.ExecutionResult != nil {
		t.overallInfo.gas += ctx.ExecutionResult.GetGasUsed()
	}

	return nil
}

// PostBlock registers the completed block and may trigger the logging of an update.
func (t *blockProgressTracker) PostBlock(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	boundary := state.Block - (state.Block % t.reportFrequency)

	if state.Block-t.lastReportedBlock < t.reportFrequency {
		return nil
	}

	now := time.Now()
	overall := now.Sub(t.startOfRun)
	interval := now.Sub(t.startOfLastInterval)

	// quickly get a snapshot of the current overall progress
	t.lock.Lock()
	info := t.overallInfo
	t.lock.Unlock()

	disk, err := utils.GetDirectorySize(ctx.StateDbPath)
	if err != nil {
		return fmt.Errorf("cannot size of state-db (%v); %v", ctx.StateDbPath, err)
	}
	m := ctx.State.GetMemoryUsage()

	memory := uint64(0)
	if m != nil {
		memory = m.UsedBytes
	}

	intervalBlkRate := float64(t.reportFrequency) / interval.Seconds()
	intervalTxRate := float64(info.numTransactions-t.lastIntervalInfo.numTransactions) / interval.Seconds()
	intervalGasRate := float64(info.gas-t.lastIntervalInfo.gas) / interval.Seconds()
	t.lastIntervalInfo = info

	overallBlkRate := float64(state.Block-int(t.cfg.First)) / overall.Seconds()
	overallTxRate := float64(info.numTransactions) / overall.Seconds()
	overallGasRate := float64(info.gas) / overall.Seconds()

	t.log.Noticef(
		substateProgressTrackerReportFormat,
		boundary, memory, disk,
		intervalBlkRate, intervalTxRate, intervalGasRate,
		overallBlkRate, overallTxRate, overallGasRate,
	)

	if t.pub != nil {
		t.pub.Publish( map[string]any{
			"start": boundary,
			"end": boundary + t.reportFrequency,
			"memory": memory,
			"txCount": intervalTxRate,
			"gas": intervalGasRate,
			"totalTxCount": overallTxRate,
			"totalGas": overallGasRate,
			"lDisk": disk,
			"aDisk": 0,
		})
	}

	t.lastReportedBlock = boundary
	t.startOfLastInterval = now

	return nil
}
