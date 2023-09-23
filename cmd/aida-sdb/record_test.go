package main

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/executor/action_provider"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

func TestRecord_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	substate := action_provider.NewMockSubstateProvider(ctrl)
	db := state.NewMockStateDB(ctrl)
	config := &utils.Config{
		First:            1,
		Last:             1,
		SyncPeriodLength: 2,
		ChainID:          utils.MainnetChainID,
		LogLevel:         "Critical",
	}

	ctx, err := context.NewRecord(t.TempDir()+"/test-record", 1)
	if err != nil {
		t.Fatalf("cannot create new record; %v", err)
	}

	r := newRecorder(config, ctx)

	substate.EXPECT().
		Run(1, 2, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer action_provider.Consumer) error {
			consumer(action_provider.TransactionInfo{Block: 1, Transaction: 0, Substate: emptyTx}, nil)
			return nil
		})

	gomock.InOrder(
		db.EXPECT().BeginBlock(uint64(1)),
		// anything else is called on a RecorderProxy
	)

	if err = record(config, substate, db, r); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

// emptyTx is a dummy substate that will be processed without crashing.
var emptyTx = &substate.Substate{
	Env: &substate.SubstateEnv{},
	Message: &substate.SubstateMessage{
		GasPrice: big.NewInt(12),
	},
	Result: &substate.SubstateResult{
		GasUsed: 1,
	},
}
