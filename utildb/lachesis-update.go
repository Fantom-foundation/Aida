package utildb

import (
	"context"
	"fmt"

	"github.com/Fantom-foundation/Aida/executor/transaction/substate_transaction"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// address of sfc contract in Hex
const sfcAddrHex = "0xFC00FACE00000000000000000000000000000000"

// LoadOperaWorldState loads opera initial world state from worldstate-db as SubstateAlloc
func LoadOperaWorldState(path string) (substate.SubstateAlloc, error) {
	worldStateDB, err := snapshot.OpenStateDB(path)
	if err != nil {
		return nil, err
	}
	defer snapshot.MustCloseStateDB(worldStateDB)
	opera, err := worldStateDB.ToSubstateAlloc(context.Background())
	return opera, err
}

// CreateLachesisWorldState creates update-set from block 0 to the last lachesis block
func CreateLachesisWorldState(cfg *utils.Config) (substate.SubstateAlloc, error) {
	lachesisLastBlock := utils.FirstOperaBlock - 1
	lachesis, _, err := utils.GenerateUpdateSet(0, lachesisLastBlock, cfg)
	if err != nil {
		return nil, err
	}
	// remove deleted accounts
	if err := utils.DeleteDestroyedAccountsFromWorldState(substate_transaction.NewOldSubstateAlloc(lachesis), cfg, lachesisLastBlock); err != nil {
		return nil, err
	}
	return lachesis, nil
}

// FixSfcContract reset lachesis's storage keys to zeros while keeping opera keys
func FixSfcContract(lachesis substate.SubstateAlloc, transition *substate.Substate) error {
	sfcAddr := common.HexToAddress(sfcAddrHex)
	lachesisSfc, hasLachesisSfc := lachesis[sfcAddr]
	_, hasTransitionSfc := transition.OutputAlloc[sfcAddr]

	if hasLachesisSfc && hasTransitionSfc {
		// insert keys with zero value to the transition substate if
		// the keys doesn't appear on opera
		for key := range lachesisSfc.Storage {
			if _, found := transition.OutputAlloc[sfcAddr].Storage[key]; !found {
				transition.OutputAlloc[sfcAddr].Storage[key] = common.Hash{}
			}
		}
	} else {
		return fmt.Errorf("the SFC contract is not found.")
	}
	return nil
}
