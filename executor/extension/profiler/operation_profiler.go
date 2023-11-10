package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/utils/analytics"
	"github.com/jedib0t/go-pretty/v6/table"
)

const sqlite3_InsertIntoOperations = `
	INSERT INTO operations(
		start, end, opId, opName, count, sum, mean, std, variance, skewness, kurtosis, min, max
	) VALUES ( 
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? 
	)
`

// MakeOperationProfiler creates a executor.Extension that records Operation profiling
func MakeOperationProfiler[T any](cfg *utils.Config) executor.Extension[T] {
	if !cfg.Profile {
		return extension.NilExtension[T]{}
	}

	ops := operation.CreateIdLabelMap()
	p := &operationProfiler[T]{
		cfg:      cfg,
		ops:      ops,
		anlt:     analytics.NewIncrementalAnalytics(len(ops)),
		ps:       utils.NewPrinters(),
		interval: utils.NewInterval(cfg.First, cfg.Last, cfg.ProfileInterval),
		log:      logger.NewLogger(cfg.LogLevel, "Operation Profiler"),
	}

	p.ps.AddPrintToConsole(func() string { return p.prettyTable().Render() })
	p.ps.AddPrintToFile(cfg.ProfileFile, func() string { return p.prettyTable().RenderCSV() })
	p.ps.AddPrintToSqlite3(cfg.ProfileSqlite3, sqlite3_InsertIntoOperations, p.insertIntoOperations)

	return p
}

type operationProfiler[T any] struct {
	extension.NilExtension[T]
	cfg      *utils.Config
	ops      map[byte]string
	anlt     *analytics.IncrementalAnalytics
	ps       *utils.Printers
	interval *utils.Interval
	log      logger.Logger
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func (p *operationProfiler[T]) prettyTable() table.Writer {
	t := table.NewWriter()

	totalCount := uint64(0)
	totalSum := 0.0

	t.AppendHeader(table.Row{
		"op", "first", "last", "n", "sum(us)", "mean(us)", "std(us)", "min(us)", "max(us)",
	})
	for opId, stat := range p.anlt.Iterate() {
		totalCount += stat.GetCount()
		totalSum += stat.GetSum()

		t.AppendRow(table.Row{
			p.ops[byte(opId)],
			p.interval.Start(),
			p.interval.End(),
			stat.GetCount(),
			stat.GetSum() / float64(1000),
			stat.GetMean() / float64(1000),
			stat.GetStandardDeviation() / float64(1000),
			stat.GetMin() / float64(1000),
			stat.GetMax() / float64(1000),
		})
	}
	t.AppendFooter(table.Row{"total", "", "", totalCount, totalSum})

	return t
}

func (p *operationProfiler[T]) insertIntoOperations() [][]any {
	values := [][]any{{}}
	for opId, stat := range p.anlt.Iterate() {
		value := []any{
			p.interval.Start(),
			p.interval.End(),
			opId,
			p.ops[byte(opId)],
			stat.GetCount(),
			stat.GetSum() / float64(1000),
			stat.GetMean() / float64(1000),
			stat.GetStandardDeviation() / float64(1000),
			stat.GetVariance() / float64(1000),
			stat.GetSkewness() / float64(1000),
			stat.GetKurtosis() / float64(1000),
			stat.GetMin() / float64(1000),
			stat.GetMax() / float64(1000),
		}
		values = append(values, value)
	}
	return values
}

func (p *operationProfiler[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	ctx.State = proxy.NewProfilerProxy(ctx.State, p.anlt, p.cfg.LogLevel)
	return nil
}

func (p *operationProfiler[T]) PreBlock(state executor.State[T], _ *executor.Context) error {
	if uint64(state.Block) > p.interval.End() {
		p.ps.Print()
		p.interval.Next()
		p.anlt.Reset()
	}
	return nil
}

func (p *operationProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.ps.Print()
	p.ps.Close()
	return nil
}
