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
	if !bp.Cfg.Profile {
		return nil
	}

	bp.Db, ext.stats = proxy.NewProfilerProxy(bp.Db, bp.Cfg.ProfileFile, bp.Cfg.LogLevel)
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
	if !bp.Cfg.Profile {
		return nil
	}

	if err := ext.stats.PrintProfiling(bp.Cfg.First, bp.Cfg.Last); err != nil {
		bp.Log.Warningf("cannot print profiling stats; %v", err)
	}

	return nil
}

func (ext *ProxyProfilerExtension) Exit(bp *BlockProcessor) error {
	return nil
}
