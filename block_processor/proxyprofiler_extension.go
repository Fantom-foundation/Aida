package blockprocessor

import (
	"github.com/Fantom-foundation/Aida/state/proxy"
	"github.com/Fantom-foundation/Aida/tracer/profile"
)

// ProxyProfilerExtension implements usage of Profiler for block processing tools
type ProxyProfilerExtension struct {
	ProcessorExtensions
	stats *profile.Stats
}

func NewProxyProfilerExtension() *ProxyProfilerExtension {
	return &ProxyProfilerExtension{}
}

// Init creates a ProfilerProxy and assigns it to the BlockProcessor's db
func (ext *ProxyProfilerExtension) Init(bp *BlockProcessor) error {
	if !bp.cfg.Profile {
		return nil
	}

	bp.db, ext.stats = proxy.NewProfilerProxy(bp.db, bp.cfg.ProfileFile, bp.cfg.LogLevel)
	return nil
}

func (ext *ProxyProfilerExtension) PostPrepare(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyProfilerExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyProfilerExtension) PostBlock(bp *BlockProcessor) error {
	return nil
}

// PostProcessing prints profiling information it was enabled
func (ext *ProxyProfilerExtension) PostProcessing(bp *BlockProcessor) error {
	if !bp.cfg.Profile {
		return nil
	}

	if err := ext.stats.PrintProfiling(bp.cfg.First, bp.cfg.Last); err != nil {
		bp.log.Warningf("cannot print profiling stats; %v", err)
	}

	return nil
}

func (ext *ProxyProfilerExtension) Exit(bp *BlockProcessor) error {
	return nil
}