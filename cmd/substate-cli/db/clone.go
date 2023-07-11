package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

var CloneCommand = cli.Command{
	Action:    clone,
	Name:      "clone",
	Usage:     "Clone a given block range from src db to dst db.",
	ArgsUsage: "<srcPath> <dstPath> <blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
	},
	Description: `
The substate-cli db clone command requires four arguments:
    <srcPath> <dstPath> <blockNumFirst> <blockNumLast>
<srcPath> is the original substate database to read the information.
<dstPath> is the target substate database to write the information.
<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to clone.

If dstPath doesn't exist, a new substate database is created.
If dstPath exits, substates from src db are merged into dst db. Any overlapping blocks are overwrittenby the new value from src db.`,
}

func clone(ctx *cli.Context) error {
	var err error

	if ctx.Args().Len() != 4 {
		return fmt.Errorf("substate-cli db clone command requires exactly 4 arguments")
	}

	srcPath := ctx.Args().Get(0)
	dstPath := ctx.Args().Get(1)
	first, last, rerr := utils.SetBlockRange(ctx.Args().Get(2), ctx.Args().Get(3), ctx.Int(utils.ChainIDFlag.Name))
	if rerr != nil {
		return rerr
	}

	// open src db as readonly
	srcBackend, err := rawdb.NewLevelDBDatabase(srcPath, 1024, 100, "srcDB", true)
	if err != nil {
		return fmt.Errorf("substate-cli db clone: error opening %s: %v", srcPath, err)
	}
	srcDB := substate.NewSubstateDB(srcBackend)
	defer srcDB.Close()

	// Create dst db as non-readonly
	dstBackend, err := rawdb.NewLevelDBDatabase(dstPath, 1024, 100, "dstDB", false)
	if err != nil {
		return fmt.Errorf("substate-cli db clone: error opening %s: %v", dstPath, err)
	}
	dstDB := substate.NewSubstateDB(dstBackend)
	defer dstDB.Close()

	cloneTask := func(block uint64, tx int, substate *substate.Substate, taskPool *substate.SubstateTaskPool) error {
		dstDB.PutSubstate(block, tx, substate)
		return nil
	}

	taskPool := substate.NewSubstateTaskPool("substate-cli db clone", cloneTask, uint64(first), uint64(last), ctx)
	taskPool.DB = srcDB
	err = taskPool.Execute()
	return err
}
