package replay

import (
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension/profiler"
	"github.com/Fantom-foundation/Aida/executor/extension/statedb"
	"github.com/Fantom-foundation/Aida/executor/extension/tracker"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc_iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

func RunRPCAdb(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	cfg.SrcDbReadonly = true

	rpcSource, err := executor.OpenRpcRecording(cfg, ctx)
	if err != nil {
		return err
	}

	defer rpcSource.Close()

	return run(cfg, rpcSource)
}

func run(cfg *utils.Config, provider executor.Provider[*rpc_iterator.RequestWithResponse]) error {
	var extensionList = []executor.Extension[*rpc_iterator.RequestWithResponse]{
		statedb.MakeStateDbManager[*rpc_iterator.RequestWithResponse](cfg),
		profiler.MakeCpuProfiler[*rpc_iterator.RequestWithResponse](cfg),
		tracker.MakeProgressLogger[*rpc_iterator.RequestWithResponse](cfg, 15*time.Second),
		statedb.MakeTemporaryArchivePrepper[*rpc_iterator.RequestWithResponse](),
	}

	return executor.NewExecutor(provider).Run(
		executor.Params{
			From:                   int(cfg.First),
			To:                     int(cfg.Last) + 1,
			NumWorkers:             cfg.Workers,
			ParallelismGranularity: executor.BlockLevel,
		},
		makeRPCProcessor(cfg),
		extensionList,
	)
}

func makeRPCProcessor(cfg *utils.Config) rpcProcessor {
	return rpcProcessor{
		cfg:     cfg,
		builder: new(strings.Builder),
		log:     logger.NewLogger(cfg.LogLevel, "RPC-Adb"),
	}
}

type rpcProcessor struct {
	cfg     *utils.Config
	builder *strings.Builder
	log     logger.Logger
}

func (p rpcProcessor) Process(state executor.State[*rpc_iterator.RequestWithResponse], ctx *executor.Context) error {
	res := p.execute(state.Data, ctx.Archive, state.Block)
	compareError := p.compare(comparisonData{
		block:   uint64(state.Block),
		record:  state.Data,
		StateDB: res,
	})

	if compareError != nil {
		p.log.Warning(compareError)
	}

	return nil
}

func (p rpcProcessor) execute(req *rpc_iterator.RequestWithResponse, archive state.NonCommittableStateDB, block int) *StateDBData {
	switch req.Query.MethodBase {
	case "getBalance":
		return executeGetBalance(req.Query.Params[0], archive)

	case "getTransactionCount":
		return executeGetTransactionCount(req.Query.Params[0], archive)

	case "call":
		var timestamp uint64

		// first try to extract timestamp from response
		if req.Response != nil {
			if req.Response.Timestamp != 0 {
				timestamp = uint64(time.Unix(0, int64(req.Response.Timestamp)).Unix())
			}
		} else if req.Error != nil {
			if req.Error.Timestamp != 0 {

				timestamp = uint64(time.Unix(0, int64(req.Error.Timestamp)).Unix())
			}
		}

		if timestamp == 0 {
			return nil
		}

		evm := newEVMExecutor(uint64(block), archive, p.cfg, req.Query.Params[0].(map[string]interface{}), timestamp, p.log)
		return executeCall(evm)

	case "estimateGas":
		// estimateGas is currently not suitable for rpc replay since the estimation  in geth is always calculated for current state
		// that means recorded result and result returned by StateDB are not comparable
	case "getCode":
		return executeGetCode(req.Query.Params[0], archive)

	case "getStorageAt":
		return executeGetStorageAt(req.Query.Params, archive)

	default:
		break
	}
	return nil
}

func (p rpcProcessor) compare(data comparisonData) (err *comparatorError) {
	switch data.record.Query.MethodBase {
	case "getBalance":
		err = compareBalance(data, p.builder)
	case "getTransactionCount":
		err = compareTransactionCount(data, p.builder)
	case "call":
		err = compareCall(data, p.builder)
		if err != nil && err.typ == expectedErrorGotResult && !data.isRecovered {
			data.isRecovered = true
			return p.tryRecovery(data)
		}
	case "estimateGas":
		// estimateGas is currently not suitable for replay since the estimation  in geth is always calculated for current state
		// that means recorded result and result returned by StateDB are not comparable
	case "getCode":
		err = compareCode(data, p.builder)
	case "getStorageAt":
		err = compareStorageAt(data, p.builder)
	}

	return
}
