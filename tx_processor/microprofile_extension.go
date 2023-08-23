package tx_processor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
)

const microProfileCtxSize = 5

// MicroProfileExtension manages state-db directory
type MicroProfileExtension struct {
	ProcessorExtensions
	collectors []*MicroProfilingCollector
}

type MicroProfilingCollector struct {
	stats  *vm.MicroProfileStatistic
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan struct{}
}

func NewMicroProfileExtension() *MicroProfileExtension {
	return &MicroProfileExtension{}
}

func newMicroProfilingCollectors() []*MicroProfilingCollector {
	s := make([]*MicroProfilingCollector, microProfileCtxSize)

	for i := 0; i < microProfileCtxSize; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		dcc := &MicroProfilingCollector{
			stats:  vm.NewMicroProfileStatistic(),
			ctx:    ctx,
			cancel: cancel,
			ch:     make(chan struct{}),
		}

		s[i] = dcc
	}

	return s
}

// Init creates collectors and starts them
func (ext *MicroProfileExtension) Init(tp *TxProcessor) error {
	if !tp.cfg.MicroProfiling {
		return nil
	}

	vm.MicroProfiling = true
	vm.MicroProfilingBufferSize = tp.cfg.ChannelBufferSize
	vm.MicroProfilingDB = tp.cfg.ProfilingDbName

	ext.collectors = newMicroProfilingCollectors()

	// start the collectors
	for _, coll := range ext.collectors {
		go vm.MicroProfilingCollector(coll.ctx, coll.ch, coll.stats)
	}

	return nil
}

func (ext *MicroProfileExtension) PostPrepare(tp *TxProcessor) error {
	return nil
}

// PostProcessing cancels all collectors context and merges and dumps collected stats into one single MicroProfilingDB
func (ext *MicroProfileExtension) PostProcessing(tp *TxProcessor) error {
	if !tp.cfg.MicroProfiling {
		return nil
	}

	// cancel collectors
	for i := 0; i < microProfileCtxSize; i++ {
		(ext.collectors[i].cancel)() // stop data collector
		<-(ext.collectors[i].ch)     // wait for data collector to finish
	}

	// create final stats and merge all collectors into it
	stats := vm.NewMicroProfileStatistic()
	for i := 0; i < microProfileCtxSize; i++ {
		stats.Merge(ext.collectors[i].stats)
	}

	version := fmt.Sprintf("chaind-id:%v", tp.cfg.ChainID)
	stats.Dump(version)

	tp.log.Noticef("MicroProfiling finished. You can find the stats here: %v", vm.MicroProfilingDB)

	return nil
}

func (ext *MicroProfileExtension) Exit(tp *TxProcessor) error {
	return nil
}
