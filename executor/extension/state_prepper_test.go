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

	gomock.InOrder(
		db.EXPECT().PrepareSubstate(&allocA, uint64(5)),
		db.EXPECT().PrepareSubstate(&allocB, uint64(7)),
	)

	prepper := MakeStateDbPreparator()

	prepper.PreTransaction(executor.State{
		Block:    5,
		State:    db,
		Substate: &substate.Substate{InputAlloc: allocA},
	})

	prepper.PreTransaction(executor.State{
		Block:    7,
		State:    db,
		Substate: &substate.Substate{InputAlloc: allocB},
	})
}

func TestStatePrepper_DoesNotCrashOnMissingStateOrSubstate(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := state.NewMockStateDB(ctrl)

	prepper := MakeStateDbPreparator()
	prepper.PreTransaction(executor.State{Block: 5})                                 // misses both
	prepper.PreTransaction(executor.State{Block: 5, State: db})                      // misses the substate
	prepper.PreTransaction(executor.State{Block: 5, Substate: &substate.Substate{}}) // misses the state
}
