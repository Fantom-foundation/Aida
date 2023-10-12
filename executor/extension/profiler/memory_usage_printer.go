package profiler

import (
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeMemoryUsagePrinter creates an executor.Extension that prints memory breakdown if enabled.
func MakeMemoryUsagePrinter(config *utils.Config) executor.Extension {
	if !config.MemoryBreakdown {
		return extension.NilExtension{}
	}

	log := logger.NewLogger(config.LogLevel, "Memory-Usage-Printer")
	return makeMemoryUsagePrinter(config, log)
}

func makeMemoryUsagePrinter(config *utils.Config, log logger.Logger) executor.Extension {
	return &memoryUsagePrinter{
		log:    log,
		config: config,
	}
}

type memoryUsagePrinter struct {
	extension.NilExtension
	log    logger.Logger
	config *utils.Config
}

func (p *memoryUsagePrinter) PreRun(_ executor.State, ctx *executor.Context) error {
	utils.MemoryBreakdown(ctx.State, p.config, p.log)
	return nil
}

func (p *memoryUsagePrinter) PostRun(_ executor.State, ctx *executor.Context, _ error) error {
	utils.MemoryBreakdown(ctx.State, p.config, p.log)
	return nil
}
