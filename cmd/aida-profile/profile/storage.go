// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package profile

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// GetStorageUpdateSizeCommand returns changes in storage size by transactions in the specified block range
var GetStorageUpdateSizeCommand = cli.Command{
	Action:    getStorageUpdateSizeAction,
	Name:      "storage-size",
	Usage:     "returns changes in storage size by transactions in the specified block range",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The util-db storage-size command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to vm transactions.

Output log format: (block, timestamp, transaction, account, storage update size, storage size in input substate, storage size in output substate)`,
}

// computeStorageSize computes the number of non-zero storage entries
func computeStorageSizes(inUpdateSet map[common.Hash]common.Hash, outUpdateSet map[common.Hash]common.Hash) (int64, uint64, uint64) {
	deltaSize := int64(0)
	inUpdateSize := uint64(0)
	outUpdateSize := uint64(0)
	wordSize := uint64(32) //bytes
	for address, outValue := range outUpdateSet {
		if inValue, found := inUpdateSet[address]; found {
			if (inValue == common.Hash{} && outValue != common.Hash{}) {
				// storage increases by one new cell
				// (cell is empty in in-storage)
				deltaSize++
			} else if (inValue != common.Hash{} && outValue == common.Hash{}) {
				// storage shrinks by one new cell
				// (cell is empty in out-storage)
				deltaSize--
			}
		} else {
			// storage increases by one new cell
			// (cell is not found in in-storage but found in out-storage)
			if (outValue != common.Hash{}) {
				deltaSize++
			}
		}
		// compute update size
		if (outValue != common.Hash{}) {
			outUpdateSize++
		}
	}
	for address, inValue := range inUpdateSet {
		if _, found := outUpdateSet[address]; !found {
			// storage shrinks by one cell
			// (The cell does not exist for an address in in-storage)
			if (inValue != common.Hash{}) {
				deltaSize--
			}
		}
		if (inValue != common.Hash{}) {
			inUpdateSize++
		}
	}
	return deltaSize * int64(wordSize), inUpdateSize * wordSize, outUpdateSize * wordSize
}

// getStorageUpdateSizeTask replays storage access of accounts in each transaction
func getStorageUpdateSizeTask(block uint64, tx int, st *substate.Substate, taskPool *substate.SubstateTaskPool) error {

	timestamp := st.Env.Timestamp
	for wallet, outputAccount := range st.OutputAlloc {
		var (
			deltaSize     int64
			inUpdateSize  uint64
			outUpdateSize uint64
		)
		// account exists in both input substate and output substate
		if inputAccount, found := st.InputAlloc[wallet]; found {
			deltaSize, inUpdateSize, outUpdateSize = computeStorageSizes(inputAccount.Storage, outputAccount.Storage)
			// account exists in output substate but not input substate
		} else {
			deltaSize, inUpdateSize, outUpdateSize = computeStorageSizes(map[common.Hash]common.Hash{}, outputAccount.Storage)
		}
		fmt.Printf("metric: %v,%v,%v,%v,%v,%v,%v\n", block, timestamp, tx, wallet.Hex(), deltaSize, inUpdateSize, outUpdateSize)
	}
	// account exists in input substate but not output substate
	for wallet, inputAccount := range st.InputAlloc {
		if _, found := st.OutputAlloc[wallet]; !found {
			deltaSize, inUpdateSize, outUpdateSize := computeStorageSizes(inputAccount.Storage, map[common.Hash]common.Hash{})
			fmt.Printf("metric: %v,%v,%v,%v,%v,%v,%v\n", block, timestamp, tx, wallet.Hex(), deltaSize, inUpdateSize, outUpdateSize)
		}
	}
	return nil
}

// func getStorageUpdateSizeAction for vm-storage command
func getStorageUpdateSizeAction(ctx *cli.Context) error {
	var err error

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "Substate Replay")

	log.Infof("chain-id: %v\n", cfg.ChainID)

	substate.SetSubstateDb(cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	taskPool := substate.NewSubstateTaskPool("aida-vm storage", getStorageUpdateSizeTask, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()
	return err
}
