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
		&substate.SubstateDbFlag,
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

	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	taskPool := substate.NewSubstateTaskPool("aida-vm storage", getCodeSizeTask, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()
	return err
}
