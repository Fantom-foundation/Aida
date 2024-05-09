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

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// GetCodeSizeCommand reports code size and nonce of smart contracts in the specified block range
var GetCodeSizeCommand = cli.Command{
	Action:    getCodeSizeAction,
	Name:      "code-size",
	Usage:     "reports code size and nonce of smart contracts in the specified block range",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
	},
	Description: `
The aida-profile code-size command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.

Output log format: (block, timestamp, transaction, account, code size, nonce, transaction type)`,
}

func GetTxType(to *common.Address, alloc substate.SubstateAlloc) string {
	if to == nil {
		return "create"
	}
	account, hasReceiver := alloc[*to]
	if to != nil && (!hasReceiver || len(account.Code) == 0) {
		return "transfer"
	}
	if to != nil && (hasReceiver && len(account.Code) > 0) {
		return "call"
	}
	return "unknown"
}

// getCodeSizeTask returns codesize and nonce of accounts in a substate
func getCodeSizeTask(block uint64, tx int, st *substate.Substate, taskPool *substate.SubstateTaskPool) error {
	to := st.Message.To
	timestamp := st.Env.Timestamp
	txType := GetTxType(to, st.InputAlloc)
	for account, accountInfo := range st.OutputAlloc {
		fmt.Printf("metric: %v,%v,%v,%v,%v,%v,%v\n",
			block,
			timestamp,
			tx,
			account.Hex(),
			len(accountInfo.Code),
			accountInfo.Nonce,
			txType)
	}
	for account, accountInfo := range st.InputAlloc {
		if _, found := st.OutputAlloc[account]; !found {
			fmt.Printf("metric: %v,%v,%v,%v,%v,%v,%v\n",
				block,
				timestamp,
				tx,
				account.Hex(),
				len(accountInfo.Code),
				accountInfo.Nonce,
				txType)
		}
	}
	return nil
}

// func getCodeSizeAction for GetCodeSizeCommand
func getCodeSizeAction(ctx *cli.Context) error {
	var err error

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	fmt.Printf("chain-id: %v\n", cfg.ChainID)

	substate.SetSubstateDb(cfg.AidaDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	taskPool := substate.NewSubstateTaskPool("aida-vm storage", getCodeSizeTask, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()
	return err
}
