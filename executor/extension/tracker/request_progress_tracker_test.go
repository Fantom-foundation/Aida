package tracker

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestRpcProgressTrackerExtension_NoLoggerIsCreatedIfDisabled(t *testing.T) {
	cfg := &utils.Config{}
	cfg.TrackProgress = false
	ext := MakeRequestProgressTracker(cfg, testStateDbInfoFrequency)
	if _, ok := ext.(extension.NilExtension[*rpc.RequestAndResults]); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestRpcProgressTrackerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}

	ext := makeRequestProgressTracker(cfg, 6, log)

	ctx := &executor.Context{}

	gomock.InOrder(
		log.EXPECT().Noticef(rpcProgressTrackerReportFormat,
			uint64(6), // boundary
			executor.MatchRate(gomock.All(executor.Gt(5), executor.Lt(6)), "intervalTotalReqRate"),
			executor.MatchRate(gomock.All(executor.Gt(62), executor.Lt(63)), "intervalGasRate"),
			executor.MatchRate(gomock.All(executor.Gt(5), executor.Lt(6)), "overallTotalReqRate"),
			executor.MatchRate(gomock.All(executor.Gt(62), executor.Lt(63)), "overallGasRate"),
		),
		log.EXPECT().Noticef(rpcProgressTrackerReportFormat,
			uint64(12), // boundary
			executor.MatchRate(gomock.All(executor.Gt(5), executor.Lt(6)), "intervalTotalReqRate"),
			executor.MatchRate(gomock.All(executor.Gt(62), executor.Lt(63)), "intervalGasRate"),
			executor.MatchRate(gomock.All(executor.Gt(5), executor.Lt(6)), "overallTotalReqRate"),
			executor.MatchRate(gomock.All(executor.Gt(62), executor.Lt(63)), "overallGasRate"),
		),
	)

	ext.PreRun(executor.State[*rpc.RequestAndResults]{}, ctx)

	ctx.ExecutionResult = rpc.NewResult(new(big.Int).Bytes(), nil, 10)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)

	ctx.ExecutionResult = rpc.NewResult(nil, errors.New("test error"), 11)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)
	time.Sleep(500 * time.Millisecond)

	ctx.ExecutionResult = rpc.NewResult(new(big.Int).Bytes(), nil, 10)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)

	ctx.ExecutionResult = rpc.NewResult(nil, errors.New("test error"), 11)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)
	time.Sleep(500 * time.Millisecond)

	ctx.ExecutionResult = rpc.NewResult(new(big.Int).Bytes(), nil, 10)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)

	ctx.ExecutionResult = rpc.NewResult(nil, errors.New("test error"), 11)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)

	ctx.ExecutionResult = rpc.NewResult(new(big.Int).Bytes(), nil, 10)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)

	ctx.ExecutionResult = rpc.NewResult(nil, errors.New("test error"), 11)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)
	time.Sleep(500 * time.Millisecond)

	ctx.ExecutionResult = rpc.NewResult(new(big.Int).Bytes(), nil, 10)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)

	ctx.ExecutionResult = rpc.NewResult(nil, errors.New("test error"), 11)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)
	time.Sleep(500 * time.Millisecond)

	ctx.ExecutionResult = rpc.NewResult(new(big.Int).Bytes(), nil, 10)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)

	ctx.ExecutionResult = rpc.NewResult(nil, errors.New("test error"), 11)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)

}

func TestRpcProgressTrackerExtension_FirstLoggingIsIgnored(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.First = 4

	ext := makeRequestProgressTracker(cfg, testStateDbInfoFrequency, log)

	ctx := &executor.Context{State: db}

	ext.PreRun(executor.State[*rpc.RequestAndResults]{
		Block:       4,
		Transaction: 0,
	}, ctx)

	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{
		Block:       4,
		Transaction: 0,
		Data:        validReq,
	}, ctx)

}

var validReq = &rpc.RequestAndResults{
	Query: &rpc.Body{
		Version:    "2.0",
		ID:         json.RawMessage{1},
		Params:     []interface{}{"0x0000000000000000000000000000000000000000", "0x2"},
		Method:     "eth_getBalance",
		Namespace:  "eth",
		MethodBase: "getBalance",
	},
	Response: &rpc.Response{
		Version:   "2.0",
		ID:        json.RawMessage{1},
		BlockID:   10,
		Timestamp: 10,
	},
}

var errReq = &rpc.RequestAndResults{
	Query: &rpc.Body{
		Version:    "2.0",
		ID:         json.RawMessage{1},
		Params:     []interface{}{"0x0000000000000000000000000000000000000000", "0x2"},
		Method:     "eth_getBalance",
		Namespace:  "eth",
		MethodBase: "getBalance",
	},
	Error: &rpc.ErrorResponse{
		Version:   "2.0",
		Id:        json.RawMessage{1},
		BlockID:   11,
		Timestamp: 11,
		Error: rpc.ErrorMessage{
			Code:    -1000,
			Message: "test error",
		},
		Payload: nil,
	},
}
