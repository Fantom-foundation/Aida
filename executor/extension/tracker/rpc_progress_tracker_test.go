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
	ext := MakeRpcProgressTracker(cfg, testStateDbInfoFrequency)
	if _, ok := ext.(extension.NilExtension[*rpc.RequestAndResults]); !ok {
		t.Errorf("Logger is enabled although not set in configuration")
	}

}

func TestRpcProgressTrackerExtension_LoggingHappens(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}

	ext := makeRpcProgressTracker(cfg, 6, log)

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

	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)
	time.Sleep(500 * time.Millisecond)

	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)
	time.Sleep(500 * time.Millisecond)

	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)

	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)
	time.Sleep(500 * time.Millisecond)

	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)
	time.Sleep(500 * time.Millisecond)

	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: validReq}, ctx)
	ext.PostTransaction(executor.State[*rpc.RequestAndResults]{Data: errReq}, ctx)

}

func TestRpcProgressTrackerExtension_FirstLoggingIsIgnored(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	cfg := &utils.Config{}
	cfg.First = 4

	ext := makeRpcProgressTracker(cfg, testStateDbInfoFrequency, log)

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
	StateDB: &rpc.StateDBData{
		Result:      new(big.Int),
		Error:       nil,
		IsRecovered: false,
		GasUsed:     10,
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
	StateDB: &rpc.StateDBData{
		Result:      nil,
		Error:       errors.New("test error"),
		IsRecovered: false,
		GasUsed:     11,
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
