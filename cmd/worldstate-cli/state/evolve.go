package state

import (
	"fmt"
	"time"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// CmdEvolveState evolves state of World State database to given target block by using substateDB data about accounts
var CmdEvolveState = cli.Command{
	Action:      evolveState,
	Name:        "evolve",
	Aliases:     []string{"e"},
	Usage:       "Evolves world state snapshot database into selected target block.",
	Description: `The evolve evolves state of stored accounts in world state snapshot database.`,
	ArgsUsage:   "<block> <substatedir> <workers>",
	Flags: []cli.Flag{
		&utils.TargetBlockFlag,
		&substate.SubstateFlag,
		&utils.ValidateFlag,
		&substate.WorkersFlag,
	},
}

// evolveState dumps state from given EVM trie into an output account-state database
func evolveState(ctx *cli.Context) error {
	// make config
	cfg, err := utils.NewConfig(ctx, utils.LastBlockArg)
	if err != nil {
		return err
	}

	// try to open state DB
	stateDB, err := snapshot.OpenStateDB(cfg.WorldStateDb)
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(stateDB)

	// try to open sub state DB
	substate.SetSubstateDirectory(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	// make logger
	log := utils.NewLogger(cfg.LogLevel, "evolve")

	startBlock, targetBlock, err := getEvolutionBlockRange(cfg, stateDB, log)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("starting evolution from block", startBlock, "target block", targetBlock)

	// logging InputSubstate inconsistencies
	var validateLog func(error)
	if cfg.ValidateWorldState {
		validateLog = factoryValidatorLogger(log)
	}

	// call evolveState with prepared arguments
	finalBlock, err := snapshot.EvolveState(stateDB, startBlock, targetBlock, cfg.Workers, factoryMakeLogger(startBlock, targetBlock, log), validateLog)
	if err != nil {
		log.Errorf("unable to EvolveState; %s", err.Error())
	}

	// if evolution to desired state didn't complete successfully
	if finalBlock != targetBlock {
		log.Warningf("last processed block was %d, substateDB didn't contain data for other blocks till target %d", finalBlock, targetBlock)
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
func getEvolutionBlockRange(cfg *utils.Config, stateDB *snapshot.StateDB, log *logging.Logger) (uint64, uint64, error) {
	// evolution until given target block
	targetBlock := cfg.TargetBlock

	if targetBlock == 0 {
		return 0, 0, fmt.Errorf("supplied target block can't be %d", targetBlock)
	}

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
			log.Infof("evolving #%d ; until #%d ; %d blocks left", blk, end, end-blk)
		default:
		}
	}
}

// factoryValidatorLogger creates logging function with runtime context.
func factoryValidatorLogger(log *logging.Logger) func(error) {
	return func(err error) {
		log.Warningf("%v", err)

	}
}
