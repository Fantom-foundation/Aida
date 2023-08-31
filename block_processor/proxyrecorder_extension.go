package blockprocessor

import (
	"github.com/Fantom-foundation/Aida/state/proxy"
	traceCtx "github.com/Fantom-foundation/Aida/tracer/context"
)

// ProxyRecorderExtension implements usage of RecorderProxy for block processing tools
type ProxyRecorderExtension struct {
	ProcessorExtensions
	rCtx *traceCtx.Record
}

func NewProxyRecorderExtension() *ProxyRecorderExtension {
	return &ProxyRecorderExtension{}
}

// Init creates a RecorderProxy and assigns it to the BlockProcessor's db
func (ext *ProxyRecorderExtension) Init(bp *BlockProcessor) error {
	var err error

	if !bp.Cfg.Trace {
		return nil
	}

	ext.rCtx, err = traceCtx.NewRecord(bp.Cfg.TraceFile, bp.Cfg.First)
	if err != nil {
		return err
	}

	bp.Db = proxy.NewRecorderProxy(bp.Db, ext.rCtx)

	return nil
}

func (ext *ProxyRecorderExtension) PostPrepare(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyRecorderExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyRecorderExtension) PostBlock(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyRecorderExtension) PostProcessing(bp *BlockProcessor) error {
	return nil
}

// Exit makes sure the trace Context is closed gracefully
func (ext *ProxyRecorderExtension) Exit(bp *BlockProcessor) error {
	ext.rCtx.Close()
	return nil
}
