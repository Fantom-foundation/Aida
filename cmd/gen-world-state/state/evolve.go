package state

import (
	"fmt"
	"github.com/Fantom-foundation/aida/cmd/gen-world-state/flags"
	"github.com/Fantom-foundation/aida/world-state/db/snapshot"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
	"time"
)

// CmdEvolveState evolves state of World State database to given target block by using substateDB data about accounts
var CmdEvolveState = cli.Command{
	Action:      evolveState,
	Name:        "evolve",
	Aliases:     []string{"e"},
	Usage:       "Evolves world state snapshot database into selected target block.",
	Description: `The evolve evolves state of stored accounts in world state snapshot database.`,
	ArgsUsage:   "<target> <substatedir> <workers>",
	Flags: []cli.Flag{
		&flags.TargetBlock,
		&flags.SubstateDBPath,
		&flags.Workers,
	},
}

// evolveState dumps state from given EVM trie into an output account-state database
func evolveState(ctx *cli.Context) error {
	// try to open state DB
	stateDB, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateDB)

	// try to open sub state DB
	substate.SetSubstateDirectory(ctx.Path(flags.SubstateDBPath.Name))
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// make logger
	log := Logger(ctx, "evolve")

	startBlock, targetBlock, err := getEvolutionBlockRange(ctx, stateDB, log)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("starting evolution from block", startBlock, "target block", targetBlock)

	// call evolveState with prepared arguments
	finalBlock, err := snapshot.EvolveState(stateDB, startBlock, targetBlock, ctx.Int(flags.Workers.Name), factoryMakeLogger(startBlock, targetBlock, log))

	// if evolution to desired state didn't complete successfully
	if finalBlock != targetBlock {
		log.Warning("last processed block was %d, substateDB didn't contain data for other blocks till target %d", finalBlock, targetBlock)
	}

	// insert new block number into database
	err = stateDB.PutBlockNumber(finalBlock)
	if err != nil {
		log.Errorf("unable to insert block number into db; %s", err.Error())
		return err
	}

	// log last processed block
	log.Noticef("database was successfully evolved to %d block", finalBlock)
	return nil
}

// getEvolutionBlockRange retrieves starting block for evolution
func getEvolutionBlockRange(ctx *cli.Context, stateDB *snapshot.StateDB, log *logging.Logger) (uint64, uint64, error) {
	// evolution until given target block
	targetBlock := ctx.Uint64(flags.TargetBlock.Name)

	// retrieving block number from world state database
	currentBlock, err := stateDB.GetBlockNumber()
	if err != nil {
		return 0, 0, err
	}
	log.Infof("database is currently at block %d", currentBlock)

	if currentBlock == targetBlock {
		return 0, 0, fmt.Errorf("world state database is already at target block %d", targetBlock)
	}

	if currentBlock > targetBlock {
		return 0, 0, fmt.Errorf("target block %d can't be lower than current block in database", targetBlock)
	}

	// database has already current block completed therefore starting at next block
	startBlock := currentBlock + 1

	return startBlock, targetBlock, nil
}

// factoryMakeLogger creates logging function with runtime context.
func factoryMakeLogger(start uint64, end uint64, log *logging.Logger) func(uint64) {
	// timer for printing progress
	tick := time.NewTicker(20 * time.Second)
	blkProgress := start

	return func(blk uint64) {
		if blk > blkProgress {
			diff := blk - blkProgress
			// if diff is more than 1 then at least 1 block was skipped
			if diff > 1 {
				log.Debugf("%d blocks skipped at #%d", diff-1, blkProgress+1)
			}
			blkProgress = blk
		}

		// print progress
		select {
		case <-tick.C:
			log.Infof("evolving #%d from #%d", blk, end)
		default:
		}
	}
}
