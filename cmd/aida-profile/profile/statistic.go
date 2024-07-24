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

package profile

import (
	"fmt"
	"sort"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/urfave/cli/v2"
)

// -------------------- Access Statistic Data Structure --------------------------

type AccessStatistics[T comparable] struct {
	accesses map[T]int
	log      logger.Logger
}

func newStatistics[T comparable](log logger.Logger) AccessStatistics[T] {
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
func collectStats[T comparable](dest chan<- T, extract Extractor[T], block uint64, tx int, st *substate.Substate, taskPool *db.SubstateTaskPool) error {
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

// getReferenceStatsActionWithConsumer extends the abilities of the function above by
// allowing some post-processing to be applied on the collected statistics.
func getReferenceStatsActionWithConsumer[T comparable](ctx *cli.Context, cli_command string, extract Extractor[T], consume AccessStatisticsConsumer[T]) error {
	var err error

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "Replay Substate")

	log.Infof("chain-id: %v\n", cfg.ChainID)
	// TODO this print has not been working ever since this functionality was introduced to aidaDb
	//log.Infof("contract-db: %v\n", cfg.Db)

	sdb, err := db.NewReadOnlySubstateDB(cfg.AidaDb)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %w", err)
	}

	// Start Collector.
	stats := newStatistics[T](log)
	done := make(chan int)
	refs := make(chan T, 100)
	go runStatCollector(&stats, refs, done)

	// Create per-transaction task.
	task := func(block uint64, tx int, st *substate.Substate, taskPool *db.SubstateTaskPool) error {
		return collectStats(refs, extract, block, tx, st, taskPool)
	}

	// Process all transactions in parallel, out-of-order.
	taskPool := sdb.NewSubstateTaskPool(fmt.Sprintf("aida-vm %v", cli_command), task, cfg.First, cfg.Last, ctx)
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
