package db

import (
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// SubstateDumpCommand returns content in substates in json format
var SubstateDumpCommand = cli.Command{
	Action:    substateDumpAction,
	Name:      "dump-substate",
	Usage:     "returns content in substates in json format",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SubstateDbFlag,
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

	substate.SetSubstateDb(ctx.String(substate.SubstateDbFlag.Name))
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	taskPool := substate.NewSubstateTaskPool("aida-vm dump", utildb.SubstateDumpTask, cfg.First, cfg.Last, ctx)
	err = taskPool.Execute()
	return err
}
