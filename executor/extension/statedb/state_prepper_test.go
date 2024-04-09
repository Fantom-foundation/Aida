package statedb

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/txcontext"
	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Substate/substate"
	substatetypes "github.com/Fantom-foundation/Substate/types"
	"go.uber.org/mock/gomock"
)

func TestStatePrepper_PreparesStateBeforeEachTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	allocA := substatecontext.NewTxContext(&substate.Substate{InputSubstate: substate.WorldState{substatetypes.Address{1}: &substate.Account{}}})
	allocB := substatecontext.NewTxContext(&substate.Substate{InputSubstate: substate.WorldState{substatetypes.Address{2}: &substate.Account{}}})
	ctx := &executor.Context{State: db}

	gomock.InOrder(
		db.EXPECT().PrepareSubstate(allocA.GetInputState(), uint64(5)),
		db.EXPECT().PrepareSubstate(allocB.GetInputState(), uint64(7)),
	)

	prepper := MakeStateDbPrepper()

	prepper.PreTransaction(executor.State[txcontext.TxContext]{
		Block: 5,
		Data:  allocA,
	}, ctx)

	prepper.PreTransaction(executor.State[txcontext.TxContext]{
		Block: 7,
		Data:  allocB,
	}, ctx)
}

func TestStatePrepper_DoesNotCrashOnMissingStateOrSubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	ctx := &executor.Context{State: db}

	prepper := MakeStateDbPrepper()
	prepper.PreTransaction(executor.State[txcontext.TxContext]{Block: 5}, nil)                                                           // misses both
	prepper.PreTransaction(executor.State[txcontext.TxContext]{Block: 5}, ctx)                                                           // misses the data
	prepper.PreTransaction(executor.State[txcontext.TxContext]{Block: 5, Data: substatecontext.NewTxContext(&substate.Substate{})}, nil) // misses the state
}
