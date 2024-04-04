package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/urfave/cli/v2"
)

// SubstateDumpCommand returns content in substates in json format
var SubstateDumpCommand = cli.Command{
	Action:    substateDumpAction,
	Name:      "dump-substate",
	Usage:     "returns content in substates in json format",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.WorkersFlag,
		&utils.AidaDbFlag,
	},
	Description: `
The aida-vm dump command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.`,
}

// substateDumpAction prepares config and arguments before SubstateDumpAction
func substateDumpAction(ctx *cli.Context) error {
	var err error

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	sdb, err := db.NewDefaultSubstateDB(cfg.AidaDb)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %w", err)
	}
	defer sdb.Close()

	taskPool := sdb.NewSubstateTaskPool("aida-vm dump", utildb.SubstateDumpTask, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()
	return err
}
