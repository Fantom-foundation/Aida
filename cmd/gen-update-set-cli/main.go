package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/gen-update-set-cli/updateset"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// GenUpdateSetApp data structure
var GenUpdateSetApp = cli.App{
	Action:    updateset.GenUpdateSet,
	Name:      "Aida Generate Update-set Manager",
	HelpName:  "gen-update-set",
	Usage:     "generate update-set from substate",
	Copyright: "(c) 2022 Fantom Foundation",
	ArgsUsage: "<blockNumLast> <interval>",
	Flags: []cli.Flag{
		&utils.ChainIDFlag,
		&utils.DeletedAccountDirFlag,
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&utils.UpdateDBDirFlag,
		&utils.ValidateFlag,
		&utils.WorldStateDirFlag,
	},
	Description: `
The gen-update-set command requires two arguments: <blockNumLast> <interval>

<blockNumLast> is last block of the inclusive range of blocks to generate update set.

<interval> is the block interval of writing update set to updateDB.`,
}

// main implements gen-update-set cli.
func main() {
	if err := GenUpdateSetApp.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
