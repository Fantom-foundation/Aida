package tx_processor

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/go-opera/evmcore"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/urfave/cli/v2"
)

// todo cpu profile?
// todo profile

type TxProcessor struct {
	cfg *utils.Config // configuration
	log logger.Logger // logger
	// more
	ctx        *cli.Context
	vmDuration time.Duration
}

// resetVmDuration to initial 0 state
func (tp *TxProcessor) resetVmDuration() {
	atomic.StoreInt64((*int64)(&tp.vmDuration), 0)
}

// addVmDuration adds delta to the duration
func (tp *TxProcessor) addVmDuration(delta time.Duration) {
	atomic.AddInt64((*int64)(&tp.vmDuration), (int64)(delta))
}

// getVmDuration returns the current state of duration
func (tp *TxProcessor) getVmDuration() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(&tp.vmDuration)))
}

// NewTxProcessor creates a new block processor instance
func NewTxProcessor(ctx *cli.Context, name string) (*TxProcessor, error) {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return nil, err
	}

	return &TxProcessor{
		cfg: cfg,
		log: logger.NewLogger(cfg.LogLevel, name),
		ctx: ctx,
	}, nil
}

// Prepare the processor
func (tp *TxProcessor) Prepare() error {
	evmcore.ProfileEVMCall = tp.cfg.ProfileEVMCall

	if tp.cfg.MicroProfiling {
		vm.MicroProfiling = true
		vm.MicroProfilingBufferSize = tp.cfg.ChannelBufferSize
		vm.MicroProfilingDB = tp.cfg.ProfilingDbName
	}

	if tp.cfg.BasicBlockProfiling {
		vm.BasicBlockProfiling = true
		vm.BasicBlockProfilingBufferSize = tp.cfg.ChannelBufferSize
		vm.BasicBlockProfilingDB = tp.cfg.ProfilingDbName
	}

	return nil
}

// Run the processor
func (tp *TxProcessor) Run(actions ExtensionList) error {
	var err error

	if err = actions.ExecuteExtensions("Init", tp); err != nil {
		return err
	}

	// close actions when return
	defer func() error {
		return actions.ExecuteExtensions("Exit", tp)
	}()

	tp.log.Info("Open SubstateDb")
	substate.SetSubstateDb(tp.cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	if err = tp.Prepare(); err != nil {
		return err
	}

	// call post-prepare actions
	if err = actions.ExecuteExtensions("PostPrepare", tp); err != nil {
		return err
	}

	task := func(block uint64, tx int, recording *substate.Substate, taskPool *substate.SubstateTaskPool) error {
		if err = tp.ReplayFunc(block, tx, recording); err != nil {
			return err
		}

		return nil
	}

	tp.resetVmDuration()

	taskPool := substate.NewSubstateTaskPool("aida-vm replay", task, tp.cfg.First, tp.cfg.Last, tp.ctx)
	if err = taskPool.Execute(); err != nil {
		return err
	}

	// call post-processing actions
	if err = actions.ExecuteExtensions("PostProcessing", tp); err != nil {
		return err
	}

	tp.log.Noticef("Net VM time: %v", tp.getVmDuration())
	utils.PrintEvmStatistics(tp.cfg)

	return nil
}

// ReplayFunc replays a tx substate
func (tp *TxProcessor) ReplayFunc(block uint64, tx int, recording *substate.Substate) error {
	var (
		err error
		db  state.StateDB
	)

	db, err = tp.makeStateDb(recording, block)
	if err != nil {
		return fmt.Errorf("cannot make state-db; %v", err)
	}

	runtime, err := utils.ProcessTx(db, tp.cfg, block, tx, recording)
	if err != nil {
		return fmt.Errorf("failed to process block %v, tx %v; %v", block, tx, err)
	}
	tp.addVmDuration(runtime)

	return nil
}

// makeStateDb creates either geth or in-memory StateDb. Other implementations are not yet supported
func (tp *TxProcessor) makeStateDb(recording *substate.Substate, block uint64) (state.StateDB, error) {
	var (
		err error
		db  state.StateDB
	)

	switch strings.ToLower(tp.cfg.DbImpl) {
	case "geth":
		db, err = state.MakeOffTheChainStateDB(recording.InputAlloc)
		if err != nil {
			return nil, err
		}
	case "geth-memory", "memory":
		db = state.MakeInMemoryStateDB(&recording.InputAlloc, block)
	default:
		return nil, fmt.Errorf("unsupported db type: %s", tp.cfg.DbImpl)
	}

	return db, nil
}
