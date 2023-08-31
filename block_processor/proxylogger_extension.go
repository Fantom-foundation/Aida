package blockprocessor

import "github.com/Fantom-foundation/Aida/state/proxy"

// ProxyLoggerExtension implements usage of LoggerProxy for block processing tools
type ProxyLoggerExtension struct {
	ProcessorExtensions
}

func NewProxyLoggerExtension() *ProxyLoggerExtension {
	return &ProxyLoggerExtension{}
}

// Init creates a LoggerProxy and assigns it to the BlockProcessor's db
func (ext *ProxyLoggerExtension) Init(bp *BlockProcessor) error {
	if !bp.Cfg.DbLogging {
		return nil
	}

	bp.Db = proxy.NewLoggerProxy(bp.Db, bp.Cfg.LogLevel)
	return nil
}

func (ext *ProxyLoggerExtension) PostPrepare(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyLoggerExtension) PostTransaction(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyLoggerExtension) PostBlock(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyLoggerExtension) PostProcessing(bp *BlockProcessor) error {
	return nil
}

func (ext *ProxyLoggerExtension) Exit(bp *BlockProcessor) error {
	return nil
}
