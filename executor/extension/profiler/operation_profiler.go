package profiler

import (
	"math"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/profile"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/op/go-logging"
)

// MakeOperationProfiler creates a executor.Extension that records Operation profiling
func MakeOperationProfiler[T any](cfg *utils.Config) executor.Extension[T] {
	if !cfg.Profile {
		return extension.NilExtension[T]{}
	}

	adjustedIntervalStart := cfg.First - (cfg.First % cfg.ProfileInterval)
	p := &operationProfiler[T]{
		cfg:           cfg,
		ps:            utils.NewPrinters(),
		log:           logger.NewLogger(cfg.LogLevel, "Operation Profiler"),
		intervalStart: cfg.First,
		intervalEnd:   adjustedIntervalStart + cfg.ProfileInterval,
	}

	p.ps.AddPrintToConsole(func() string { return p.prettyTable().Render() })
	p.ps.AddPrintToFile(cfg.ProfileFile, func() string { return p.prettyTable().RenderCSV() })

	return p
}

type operationProfiler[T any] struct {
	extension.NilExtension[T]
	cfg  *utils.Config
	anlt *profile.IncrementalAnalytics
	ps   *utils.Printers
	log  *logging.Logger

	intervalStart      uint64
	intervalEnd        uint64
	lastProcessedBlock uint64
}

func (p *operationProfiler[T]) prettyTable() table.Writer {
	t := table.NewWriter()

	totalCount := uint64(0)
	totalSum := 0.0

	t.AppendHeader(table.Row{
		"id", "first", "last", "n", "sum(us)", "mean(us)", "std(us)", "min(us)", "max(us)",
	})
	for opId, stat := range p.anlt.Iterate() {
		totalCount += stat.GetCount()
		totalSum += stat.GetSum()

		t.AppendRow(table.Row{
			opId,
			p.intervalStart,
			p.intervalEnd,
			stat.GetCount(),
			stat.GetSum(),
			stat.GetMean(),
			math.Sqrt(stat.GetVariance()),
			stat.GetMin(),
			stat.GetMax(),
		})
	}
	t.AppendFooter(table.Row{"total", "", "", totalCount, totalSum})

	return t
}

func (p *operationProfiler[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	ctx.State, p.anlt = proxy.NewProfilerProxy(
		ctx.State,
		p.cfg.ProfileFile,
		p.cfg.LogLevel,
	)
	return nil
}

func (p *operationProfiler[T]) PreBlock(state executor.State[T], _ *executor.Context) error {
	if uint64(state.Block) > p.intervalEnd {
		p.ps.Print()
		p.intervalStart = p.intervalEnd + 1
		p.intervalEnd = p.intervalEnd + p.cfg.ProfileInterval
		p.anlt.Reset()
	}

	return nil
}

func (p *operationProfiler[T]) PostBlock(state executor.State[T], _ *executor.Context) error {
	p.lastProcessedBlock = uint64(state.Block)
	return nil
}

func (p *operationProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.ps.Print()
	return nil
}
