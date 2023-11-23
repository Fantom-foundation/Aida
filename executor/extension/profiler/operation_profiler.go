package profiler

import (
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/utils/analytics"
	"github.com/jedib0t/go-pretty/v6/table"
)

type ProfileDepth int

const (
	IntervalLevel ProfileDepth = iota
	BlockLevel
	TransactionLevel
)

const BufferSize = 1_000_000

const (
	sqlite3_Interval_CreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS ops_interval (
			start INTEGER NOT NULL, 
			end INTEGER NOT NULL, 
			opId INTEGER NOT NULL,
			opName STRING,
			count INTEGER,
		 	sum FLOAT,
	 		mean FLOAT,
			std FLOAT,
			variance FLOAT,
			skewness FLOAT,
			kurtosis FLOAT,
			min FLOAT,
			max FLOAT,
			PRIMARY KEY (start, end, opId)
		)
	`
	sqlite3_Block_CreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS ops_block (
			blockId INTEGER NOT NULL, 
			opId INTEGER NOT NULL,
			opName STRING,
			count INTEGER,
		 	sum FLOAT,
	 		mean FLOAT,
			std FLOAT,
			variance FLOAT,
			skewness FLOAT,
			kurtosis FLOAT,
			min FLOAT,
			max FLOAT,
			PRIMARY KEY (blockId, opId)
		)
	`
	sqlite3_Transaction_CreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS ops_transaction (
			blockId INTEGER NOT NULL,
			txId INTEGER NOT NULL,
			opId INTEGER NOT NULL,
			opName STRING,
			count INTEGER,
		 	sum FLOAT,
	 		mean FLOAT,
			std FLOAT,
			variance FLOAT,
			skewness FLOAT,
			kurtosis FLOAT,
			min FLOAT,
			max FLOAT,
			PRIMARY KEY (blockId, txId, opId)
		)
	`
	sqlite3_Interval_InsertOrReplace = `
		INSERT or REPLACE INTO ops_interval (
			start, end, opId, opName, count, sum, mean, std, variance, skewness, kurtosis, min, max
		) VALUES ( 
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? 
		)
	`
	sqlite3_Block_InsertOrReplace = `
		INSERT or REPLACE INTO ops_block (
			blockId, opId, opName, count, sum, mean, std, variance, skewness, kurtosis, min, max
		) VALUES ( 
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`
	sqlite3_Transaction_InsertOrReplace = `
		INSERT or REPLACE INTO ops_transaction (
			blockId, txId, opId, opName, count, sum, mean, std, variance, skewness, kurtosis, min, max
		) VALUES ( 
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? 
		)
	`
)

// MakeOperationProfiler creates a executor.Extension that records Operation profiling
func MakeOperationProfiler[T any](cfg *utils.Config) executor.Extension[T] {

	if !cfg.Profile {
		return extension.NilExtension[T]{}
	}

	var (
		depth ProfileDepth
		ops   map[byte]string
		anlts []*analytics.IncrementalAnalytics
		ps    []*utils.Printers
	)

	depth = ProfileDepth(cfg.ProfileDepth)
	ops = operation.CreateIdLabelMap()

	// analytics are created for each depth level
	for i := 0; i < cfg.ProfileDepth+1; i++ {
		anlts = append(anlts, analytics.NewIncrementalAnalytics(len(ops)))
		ps = append(ps, utils.NewPrinters())
	}

	p := &operationProfiler[T]{
		cfg:      cfg,
		depth:    depth,
		ops:      ops,
		anlts:    anlts,
		ps:       ps,
		interval: utils.NewInterval(cfg.First, cfg.Last, cfg.ProfileInterval),
		log:      logger.NewLogger(cfg.LogLevel, "Operation Profiler"),
	}

	// Always print profiling results after each interval.
	// TODO: log to Console
	ps[IntervalLevel].AddPrinterToConsole(true, func() string { return p.prettyTable().Render() })
	ps[IntervalLevel].AddPrinterToFile(cfg.ProfileFile, func() string { return p.prettyTable().RenderCSV() })

	// At the configured level, print to file/db if the respective flags are enabled.
	p2db, _ := utils.NewPrinterToSqlite3(p.sqlite3(cfg.ProfileSqlite3, p.depth))
	p2buffer, f2db := p2db.Bufferize(BufferSize)
	ps[p.depth].AddPrinter(p2buffer)   // print to buffer at configured depth
	ps[IntervalLevel].AddPrinter(f2db) // always flush at the end of interval, end of run

	return p
}

// operationProfiler can profile at interval, block or transaction level
type operationProfiler[T any] struct {
	extension.NilExtension[T]

	// configuration
	cfg   *utils.Config
	depth ProfileDepth

	// analytics/printing
	ops   map[byte]string
	anlts []*analytics.IncrementalAnalytics
	ps    []*utils.Printers

	// where am i?
	interval                 *utils.Interval
	lastProcessedBlock       int
	lastProcessedTransaction int

	log logger.Logger
}

func (p *operationProfiler[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	// Instantiate a proxy for each level of depth
	// wrap from deepest level first
	for d := p.depth; d >= IntervalLevel; d-- {
		ctx.State = proxy.NewProfilerProxy(ctx.State, p.anlts[d], p.cfg.LogLevel)
	}
	return nil
}

func (p *operationProfiler[T]) PreBlock(state executor.State[T], _ *executor.Context) error {
	// On Interval Change -> Print and reset interval level analytics
	// Since there are blocks without transaction, change can only be detected at the beginning of the upcoming block
	if uint64(state.Block) > p.interval.End() {
		p.ps[IntervalLevel].Print()
		p.interval.Next()
		p.anlts[IntervalLevel].Reset()
	}
	return nil
}

func (p *operationProfiler[T]) PostBlock(state executor.State[T], _ *executor.Context) error {
	// On Block End -> Print and reset block level analytics
	p.lastProcessedBlock = state.Block
	if p.depth >= BlockLevel {
		p.ps[BlockLevel].Print()
		p.anlts[BlockLevel].Reset()
	}
	return nil
}

func (p *operationProfiler[T]) PostTransaction(state executor.State[T], _ *executor.Context) error {
	// On Transaction End -> Print and reset tx level analytics
	p.lastProcessedTransaction = state.Transaction
	if p.depth >= TransactionLevel {
		p.ps[TransactionLevel].Print()
		p.anlts[TransactionLevel].Reset()
	}
	return nil
}

func (p *operationProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	// Print any analytics still unprinted and clean up
	p.ps[IntervalLevel].Print()
	p.anlts[IntervalLevel].Reset() // so it's consistant with other levels
	for _, printers := range p.ps {
		printers.Close() // close all printers
	}
	return nil
}

//
// Printer-related
//

func (p *operationProfiler[T]) prettyTable() table.Writer {
	t := table.NewWriter()

	totalCount := uint64(0)
	totalSum := 0.0

	t.AppendHeader(table.Row{
		"op", "first", "last", "n", "sum(us)", "mean(us)", "std(us)", "min(us)", "max(us)",
	})
	for opId, stat := range p.anlts[IntervalLevel].Iterate() {
		totalCount += stat.GetCount()
		totalSum += stat.GetSum()

		t.AppendRow(table.Row{
			p.ops[byte(opId)],
			p.interval.Start(),
			p.interval.End(),
			stat.GetCount(),
			stat.GetSum() / float64(time.Microsecond),
			stat.GetMean() / float64(time.Microsecond),
			stat.GetStandardDeviation() / float64(time.Microsecond),
			stat.GetMin() / float64(time.Microsecond),
			stat.GetMax() / float64(time.Microsecond),
		})
	}
	t.AppendFooter(table.Row{"total", "", "", totalCount, totalSum})

	return t
}

func (p *operationProfiler[T]) sqlite3(conn string, depth ProfileDepth) (string, string, string, func() [][]any) {
	switch depth {
	case IntervalLevel:
		return conn, sqlite3_Interval_CreateTableIfNotExist, sqlite3_Interval_InsertOrReplace,
			func() [][]any {
				values := [][]any{}
				for opId, stat := range p.anlts[depth].Iterate() {
					if stat.GetCount() == 0 {
						continue
					}

					values = append(values, []any{
						p.interval.Start(),
						p.interval.End(),
						opId,
						p.ops[byte(opId)],
						stat.GetCount(),
						stat.GetSum() / float64(time.Microsecond),
						stat.GetMean() / float64(time.Microsecond),
						stat.GetStandardDeviation() / float64(time.Microsecond),
						stat.GetVariance() / float64(time.Microsecond),
						stat.GetSkewness() / float64(time.Microsecond),
						stat.GetKurtosis() / float64(time.Microsecond),
						stat.GetMin() / float64(time.Microsecond),
						stat.GetMax() / float64(time.Microsecond),
					})
				}
				return values
			}

	case BlockLevel:
		return conn, sqlite3_Block_CreateTableIfNotExist, sqlite3_Block_InsertOrReplace,
			func() [][]any {
				values := [][]any{}
				for opId, stat := range p.anlts[depth].Iterate() {
					if stat.GetCount() == 0 {
						continue
					}

					values = append(values, []any{
						p.lastProcessedBlock,
						opId,
						p.ops[byte(opId)],
						stat.GetCount(),
						stat.GetSum() / float64(time.Microsecond),
						stat.GetMean() / float64(time.Microsecond),
						stat.GetStandardDeviation() / float64(time.Microsecond),
						stat.GetVariance() / float64(time.Microsecond),
						stat.GetSkewness() / float64(time.Microsecond),
						stat.GetKurtosis() / float64(time.Microsecond),
						stat.GetMin() / float64(time.Microsecond),
						stat.GetMax() / float64(time.Microsecond),
					})
				}
				return values
			}

	case TransactionLevel:
		return conn, sqlite3_Transaction_CreateTableIfNotExist, sqlite3_Transaction_InsertOrReplace,
			func() [][]any {
				values := [][]any{}
				for opId, stat := range p.anlts[depth].Iterate() {
					if stat.GetCount() == 0 {
						continue
					}

					values = append(values, []any{
						p.lastProcessedBlock,
						p.lastProcessedTransaction,
						opId,
						p.ops[byte(opId)],
						stat.GetCount(),
						stat.GetSum() / float64(time.Microsecond),
						stat.GetMean() / float64(time.Microsecond),
						stat.GetStandardDeviation() / float64(time.Microsecond),
						stat.GetVariance() / float64(time.Microsecond),
						stat.GetSkewness() / float64(time.Microsecond),
						stat.GetKurtosis() / float64(time.Microsecond),
						stat.GetMin() / float64(time.Microsecond),
						stat.GetMax() / float64(time.Microsecond),
					})
				}
				return values
			}
	}

	return "", "", "", nil // results in printer doing nothing
}
