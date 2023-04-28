package replay

import (
	"fmt"
	"sort"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// -------------------- Access Statistic Data Structure --------------------------

type AccessStatistics[T comparable] struct {
	accesses map[T]int
	log      *logging.Logger
}

func newStatistics[T comparable](log *logging.Logger) AccessStatistics[T] {
	return AccessStatistics[T]{
		accesses: map[T]int{},
		log:      log,
	}
}

func (a *AccessStatistics[T]) RegisterAccess(reference *T) {
	a.accesses[*reference]++
}

func (a *AccessStatistics[T]) PrintSummary() {
	var count = len(a.accesses)
	var sum int64 = 0
	list := make([]int, 0, len(a.accesses))
	for _, count := range a.accesses {
		sum += int64(count)
		list = append(list, count)
	}
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })

	var prefix_sum = 0
	for i := range list {
		list[i] = prefix_sum + list[i]
		prefix_sum = list[i]
	}

	a.log.Notice("Reference frequency distribution:")
	for i := 0; i < 100; i++ {
		a.log.Infof("%d, %d", i, list[i*len(list)/100])
	}
	a.log.Infof("100, %d\n", list[len(list)-1])
	a.log.Infof("Number of targets:          %15d\n", count)
	a.log.Infof("Number of references:       %15d\n", sum)
	a.log.Infof("Average references/target:  %15.2f\n", float32(sum)/float32(count))
}

type AccessStatisticsConsumer[T comparable] func(*AccessStatistics[T])

// ----------------------------- Access Statistic Tools ---------------------------------

type TransactionInfo struct {
	block uint64
	tx    int
	st    *substate.Substate
}

type Extractor[T any] func(*TransactionInfo) []T

func runStatCollector[T comparable](stats *AccessStatistics[T], src <-chan T, done chan<- int) {
	for reference := range src {
		stats.RegisterAccess(&reference)
	}
	close(done)
}

// collectAddressStats collects statistical information on address usage.
func collectStats[T comparable](dest chan<- T, extract Extractor[T], block uint64, tx int, st *substate.Substate, taskPool *substate.SubstateTaskPool) error {
	info := TransactionInfo{
		block: block,
		tx:    tx,
		st:    st,
	}

	// Collect all references triggered by this transaction.
	accessed_references := map[T]int{}
	for _, reference := range extract(&info) {
		accessed_references[reference] = 0
	}
	// Report accessed addresses to statistics collector.
	for reference := range accessed_references {
		dest <- reference
	}
	return nil
}

// getReferenceStatsAction a generic utility to collect access statistics from recorded
// substate data.
func getReferenceStatsAction[T comparable](ctx *cli.Context, cli_command string, extract Extractor[T]) error {
	return getReferenceStatsActionWithConsumer(ctx, cli_command, extract, func(*AccessStatistics[T]) {})
}

// getReferenceStatsActionWithConsumer extends the abilitities of the function above by
// allowing some post-processing to be applied on the collected statistics.
func getReferenceStatsActionWithConsumer[T comparable](ctx *cli.Context, cli_command string, extract Extractor[T], consume AccessStatisticsConsumer[T]) error {
	var err error

	log := utils.NewLogger(ctx.String(utils.LogLevelFlag.Name), "Replay Substate")

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("substate-cli %v command requires exactly 2 arguments", cli_command)
	}

	chainID = ctx.Int(ChainIDFlag.Name)
	log.Infof("chain-id: %v\n", chainID)
	log.Infof("git-date: %v\n", gitDate)
	log.Infof("git-commit: %v\n", gitCommit)
	log.Infof("contract-db: %v\n", ContractDB)

	first, last, argErr := utils.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1))
	if argErr != nil {
		return argErr
	}

	substate.SetSubstateDirectory(ctx.String(substate.SubstateDirFlag.Name))
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// Start Collector.
	stats := newStatistics[T](log)
	done := make(chan int)
	refs := make(chan T, 100)
	go runStatCollector(&stats, refs, done)

	// Create per-transaction task.
	task := func(block uint64, tx int, st *substate.Substate, taskPool *substate.SubstateTaskPool) error {
		return collectStats(refs, extract, block, tx, st, taskPool)
	}

	// Process all transactions in parallel, out-of-order.
	taskPool := substate.NewSubstateTaskPool(fmt.Sprintf("substate-cli %v", cli_command), task, first, last, ctx)
	err = taskPool.Execute()
	if err != nil {
		return err
	}

	// Signal the end of the processed addresses.
	close(refs)

	// Wait for the collector to finish.
	for {
		if _, open := <-done; !open {
			break
		}
	}

	// Print the statistics.
	fmt.Printf("\n\n-------- Summary: ----------\n")
	stats.PrintSummary()
	fmt.Printf("----------------------------\n")
	consume(&stats)
	return nil
}
