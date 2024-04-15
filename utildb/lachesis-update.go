// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utildb

import (
	"fmt"

	substatecontext "github.com/Fantom-foundation/Aida/txcontext/substate"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// address of sfc contract in Hex
const sfcAddrHex = "0xFC00FACE00000000000000000000000000000000"

// LoadOperaWorldState loads opera initial world state from worldstate-db as SubstateAlloc
func LoadOperaWorldState(path string) (*substate.SubstateAlloc, error) {
	//TODO: the initial world state is expected to be in updateset format
	udb, err := substate.OpenUpdateDB(path)
	if err != nil {
		return nil, err
	}
	defer udb.Close()

	return udb.GetUpdateSet(utils.FirstOperaBlock), nil
}

// CreateLachesisWorldState creates update-set from block 0 to the last lachesis block
func CreateLachesisWorldState(cfg *utils.Config) (substate.SubstateAlloc, error) {
	lachesisLastBlock := utils.FirstOperaBlock - 1
	lachesis, _, err := utils.GenerateUpdateSet(0, lachesisLastBlock, cfg)
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
