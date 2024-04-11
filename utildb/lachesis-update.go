package utildb

import (
	"context"
	"fmt"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/Substate/substate"
	substatetypes "github.com/Fantom-foundation/Substate/types"
)

// address of sfc contract in Hex
const sfcAddrHex = "0xFC00FACE00000000000000000000000000000000"

// LoadOperaWorldState loads opera initial world state from worldstate-db as WorldState
func LoadOperaWorldState(path string) (substate.WorldState, error) {
	worldStateDB, err := snapshot.OpenStateDB(path)
	if err != nil {
		return nil, err
	}
	defer snapshot.MustCloseStateDB(worldStateDB)
	opera, err := worldStateDB.ToWorldState(context.Background())
	return opera, err
}

// CreateLachesisWorldState creates update-set from block 0 to the last lachesis block
func CreateLachesisWorldState(cfg *utils.Config, aidaDb db.BaseDB) (substate.WorldState, error) {
	lachesisLastBlock := utils.FirstOperaBlock - 1
	lachesis, _, err := utils.GenerateUpdateSet(0, lachesisLastBlock, cfg, aidaDb)
	if err != nil {
		return nil, err
	}
	// remove deleted accounts
	if err := utils.DeleteDestroyedAccountsFromWorldState(substatecontext.NewWorldState(lachesis), cfg, lachesisLastBlock); err != nil {
		return nil, err
	}
	return lachesis, nil
}

// FixSfcContract reset lachesis's storage keys to zeros while keeping opera keys
func FixSfcContract(lachesis substate.WorldState, transition *substate.Substate) error {
	sfcAddr := substatetypes.HexToAddress(sfcAddrHex)
	lachesisSfc, hasLachesisSfc := lachesis[sfcAddr]
	_, hasTransitionSfc := transition.OutputSubstate[sfcAddr]

	if hasLachesisSfc && hasTransitionSfc {
		// insert keys with zero value to the transition substate if
		// the keys doesn't appear on opera
		for key := range lachesisSfc.Storage {
			if _, found := transition.OutputSubstate[sfcAddr].Storage[key]; !found {
				transition.OutputSubstate[sfcAddr].Storage[key] = substatetypes.Hash{}
			}
		}
	} else {
		return fmt.Errorf("the SFC contract is not found.")
	}
	return nil
}
