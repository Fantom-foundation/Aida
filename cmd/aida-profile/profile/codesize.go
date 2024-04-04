package profile

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/Substate/substate"
	"github.com/Fantom-foundation/Substate/types"
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
		&utils.WorkersFlag,
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

func GetTxType(to *common.Address, alloc substate.WorldState) string {
	if to == nil {
		return "create"
	}
	account, hasReceiver := alloc[types.Address(*to)]
	if to != nil && (!hasReceiver || len(account.Code) == 0) {
		return "transfer"
	}
	if to != nil && (hasReceiver && len(account.Code) > 0) {
		return "call"
	}
	return "unknown"
}

// getCodeSizeTask returns codesize and nonce of accounts in a substate
func getCodeSizeTask(block uint64, tx int, st *substate.Substate, taskPool *db.SubstateTaskPool) error {
	to := st.Message.To
	timestamp := st.Env.Timestamp
	txType := GetTxType((*common.Address)(to), st.InputSubstate)
	for account, accountInfo := range st.OutputSubstate {
		fmt.Printf("metric: %v,%v,%v,%v,%v,%v,%v\n",
			block,
			timestamp,
			tx,
			account.String(),
			len(accountInfo.Code),
			accountInfo.Nonce,
			txType)
	}
	for account, accountInfo := range st.InputSubstate {
		if _, found := st.OutputSubstate[account]; !found {
			fmt.Printf("metric: %v,%v,%v,%v,%v,%v,%v\n",
				block,
				timestamp,
				tx,
				account.String(),
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

	sdb, err := db.NewDefaultSubstateDB(cfg.AidaDb)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %w", err)
	}

	taskPool := sdb.NewSubstateTaskPool("aida-vm storage", getCodeSizeTask, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()
	return err
}
