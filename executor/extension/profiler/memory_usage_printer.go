package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeMemoryUsagePrinter creates an executor.Extension that prints memory breakdown if enabled.
func MakeMemoryUsagePrinter[T any](cfg *utils.Config) executor.Extension[T] {
	if !cfg.MemoryBreakdown {
		return extension.NilExtension[T]{}
	}

	log := logger.NewLogger(cfg.LogLevel, "Memory-Usage-Printer")
	return makeMemoryUsagePrinter[T](cfg, log)
}

func makeMemoryUsagePrinter[T any](cfg *utils.Config, log logger.Logger) executor.Extension[T] {
	return &memoryUsagePrinter[T]{
		log: log,
		cfg: cfg,
	}
}

type memoryUsagePrinter[T any] struct {
	extension.NilExtension[T]
	log logger.Logger
	cfg *utils.Config
}

func (p *memoryUsagePrinter[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	utils.MemoryBreakdown(ctx.State, p.cfg, p.log)
	return nil
}

func (p *memoryUsagePrinter[T]) PostRun(_ executor.State[T], ctx *executor.Context, _ error) error {
	utils.MemoryBreakdown(ctx.State, p.cfg, p.log)
	return nil
}
