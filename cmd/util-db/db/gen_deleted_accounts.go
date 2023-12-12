package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

var GenDeletedAccountsCommand = cli.Command{
	Action:    genDeletedAccountsAction,
	Name:      "gen-deleted-accounts",
	Usage:     "executes full state transitions and record suicided accounts",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SubstateDbFlag,
		&utils.ChainIDFlag,
		&utils.DeletionDbFlag,
		&utils.CpuProfileFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The util-db gen-deleted-accounts command requires two arguments:
<blockNumFirst> <blockNumLast>
<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.`,
}

// genDeletedAccountsAction prepares config and arguments before GenDeletedAccountsAction
func genDeletedAccountsAction(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	if cfg.DeletionDb == "" {
		return fmt.Errorf("you need to specify where you want deletion-db to save (--deletion-db)")
	}

	if cfg.SubstateDb == "" {
		return fmt.Errorf("you need to specify path to existing substate (--substate-db)")
	}

	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	ddb, err := substate.OpenDestroyedAccountDB(cfg.DeletionDb)
	if err != nil {
		return err
	}
	defer ddb.Close()

	return utildb.GenDeletedAccountsAction(cfg, ddb, cfg.First, cfg.Last)
}
