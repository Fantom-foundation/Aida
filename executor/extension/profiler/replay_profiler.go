package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeReplayProfiler creates executor.Extension that prints profile statistics
func MakeReplayProfiler[T any](cfg *utils.Config, rCtx *context.Replay) executor.Extension[T] {
	if !cfg.Profile {
		return extension.NilExtension[T]{}
	}

	return &replayProfiler[T]{
		cfg:  cfg,
		rCtx: rCtx,
	}
}

type replayProfiler[T any] struct {
	extension.NilExtension[T]
	cfg  *utils.Config
	rCtx *context.Replay
}

func (p *replayProfiler[T]) PostRun(executor.State[T], *executor.Context, error) error {
	p.rCtx.Stats.FillLabels(operation.CreateIdLabelMap())
	if err := p.rCtx.Stats.PrintProfiling(p.cfg.First, p.cfg.Last); err != nil {
		return err
	}

	return nil
}
