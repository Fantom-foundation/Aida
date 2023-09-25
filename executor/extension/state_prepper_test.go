package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestStatePrepper_PreparesStateBeforeEachTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	allocA := substate.SubstateAlloc{common.Address{1}: nil}
	allocB := substate.SubstateAlloc{common.Address{2}: nil}
	context := &executor.Context{State: db}

	gomock.InOrder(
		db.EXPECT().PrepareSubstate(&allocA, uint64(5)),
		db.EXPECT().PrepareSubstate(&allocB, uint64(7)),
	)

	prepper := MakeStateDbPreparator()

	prepper.PreAction(executor.State{
		Block:    5,
		Substate: &substate.Substate{InputAlloc: allocA},
	}, context)

	prepper.PreAction(executor.State{
		Block:    7,
		Substate: &substate.Substate{InputAlloc: allocB},
	}, context)
}

func TestStatePrepper_DoesNotCrashOnMissingStateOrSubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)
	context := &executor.Context{State: db}

	prepper := MakeStateDbPreparator()
	prepper.PreAction(executor.State{Block: 5}, nil)                                 // misses both
	prepper.PreAction(executor.State{Block: 5}, context)                             // misses the substate
	prepper.PreAction(executor.State{Block: 5, Substate: &substate.Substate{}}, nil) // misses the state
}
