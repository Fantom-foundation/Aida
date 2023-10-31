package replay

import (
	"encoding/json"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestVmSdb_TransactionsAreExecutedForCorrectRange(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	processor := executor.NewMockProcessor[*rpc.RequestAndResults](ctrl)
	ext := executor.NewMockExtension[*rpc.RequestAndResults](ctrl)
	stateDb := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	// Simulate the execution of three transactions in two blocks.
	provider.EXPECT().
		Run(10, 12, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: i, Transaction: 0, Data: emptyReqA})
				consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: i, Transaction: 0, Data: emptyReqB})
			}
			return nil
		})

	pre := ext.EXPECT().PreRun(executor.AtBlock[*rpc.RequestAndResults](10), gomock.Any())
	post := ext.EXPECT().PostRun(executor.AtBlock[*rpc.RequestAndResults](12), gomock.Any(), nil)

	gomock.InOrder(
		pre,
		stateDb.EXPECT().GetArchiveState(uint64(10)).Return(archive, nil),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](10), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](10), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](10), gomock.Any()),
		archive.EXPECT().Release(),
		post,
	)
	gomock.InOrder(
		pre,
		stateDb.EXPECT().GetArchiveState(uint64(10)).Return(archive, nil),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](10), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](10), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](10), gomock.Any()),
		archive.EXPECT().Release(),
		post,
	)
	gomock.InOrder(
		pre,
		stateDb.EXPECT().GetArchiveState(uint64(11)).Return(archive, nil),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](11), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](11), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](11), gomock.Any()),
		archive.EXPECT().Release(),
		post,
	)
	gomock.InOrder(
		pre,
		stateDb.EXPECT().GetArchiveState(uint64(11)).Return(archive, nil),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](11), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](11), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](11), gomock.Any()),
		archive.EXPECT().Release(),
		post,
	)

	cfg := &utils.Config{}
	cfg.ChainID = 250
	cfg.Workers = 4
	cfg.First = 10
	cfg.Last = 11
	if err := run(cfg, provider, processor, []executor.Extension[*rpc.RequestAndResults]{ext}, stateDb); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

var emptyReqA = &rpc.RequestAndResults{
	Query: &rpc.Body{
		Version:    "2.0",
		ID:         json.RawMessage{1},
		Params:     []interface{}{""},
		Method:     "eth_getBalance",
		Namespace:  "eth",
		MethodBase: "getBalance",
	},
	Response: &rpc.Response{
		Version: "2.0",
		ID:      json.RawMessage{1},

		BlockID:   10,
		Timestamp: 10,
		Result:    json.RawMessage{0},
		Payload:   nil,
	},
	Error:       nil,
	ParamsRaw:   nil,
	ResponseRaw: nil,
}

var emptyReqB = &rpc.RequestAndResults{
	Query: &rpc.Body{
		Version:    "2.0",
		ID:         json.RawMessage{1},
		Params:     []interface{}{""},
		Method:     "eth_getBalance",
		Namespace:  "eth",
		MethodBase: "getBalance",
	},
	Response: &rpc.Response{
		Version:   "2.0",
		ID:        json.RawMessage{1},
		BlockID:   11,
		Timestamp: 11,
		Result:    json.RawMessage{0},
		Payload:   nil,
	},
	Error:       nil,
	ParamsRaw:   nil,
	ResponseRaw: nil,
}
