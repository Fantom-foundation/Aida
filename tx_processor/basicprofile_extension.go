package tx_processor

import (
	"context"

	"github.com/ethereum/go-ethereum/core/vm"
)

const basicProfileCtxSize = 5

// BasicProfileExtension manages state-db directory
type BasicProfileExtension struct {
	ProcessorExtensions
	collectors []*BasicBlockProfilingCollector
}

type BasicBlockProfilingCollector struct {
	stats  *vm.BasicBlockProfileStatistic
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan struct{}
}

func NewBasicProfileExtension() *BasicProfileExtension {
	return &BasicProfileExtension{}
}

func newBasicProfilingCollectors() []*BasicBlockProfilingCollector {
	s := make([]*BasicBlockProfilingCollector, basicProfileCtxSize)

	for i := 0; i < basicProfileCtxSize; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		dcc := &BasicBlockProfilingCollector{
			stats:  vm.NewBasicBlockProfileStatistic(),
			ctx:    ctx,
			cancel: cancel,
			ch:     make(chan struct{}),
		}

		s[i] = dcc
	}

	return s
}

// Init creates collectors and starts them
func (ext *BasicProfileExtension) Init(tp *TxProcessor) error {
	if !tp.cfg.BasicBlockProfiling {
		return nil
	}

	vm.BasicBlockProfiling = true
	vm.BasicBlockProfilingBufferSize = tp.cfg.ChannelBufferSize
	vm.BasicBlockProfilingDB = tp.cfg.ProfilingDbName

	ext.collectors = newBasicProfilingCollectors()

	// start the collectors
	for _, coll := range ext.collectors {
		go vm.BasicBlockProfilingCollector(coll.ctx, coll.ch, coll.stats)
	}

	return nil
}

func (ext *BasicProfileExtension) PostPrepare(tp *TxProcessor) error {
	return nil
}

// PostProcessing cancels all collectors context and merges and dumps collected stats into one single MicroProfilingDB
func (ext *BasicProfileExtension) PostProcessing(tp *TxProcessor) error {
	if !tp.cfg.BasicBlockProfiling {
		return nil
	}

	// cancel collectors
	for i := 0; i < basicProfileCtxSize; i++ {
		(ext.collectors[i].cancel)() // stop data collector
		<-(ext.collectors[i].ch)     // wait for data collector to finish
	}

	// create final stats and merge all collectors into it
	stats := vm.NewBasicBlockProfileStatistic()
	for i := 0; i < basicProfileCtxSize; i++ {
		stats.Merge(ext.collectors[i].stats)
	}

	stats.Dump()

	tp.log.Noticef("BasicProfiling finished. You can find the stats here: %v", vm.MicroProfilingDB)

	return nil
}

func (ext *BasicProfileExtension) Exit(tp *TxProcessor) error {
	return nil
}
