package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

var LachesisUpdateCommand = cli.Command{
	Action: lachesisUpdate,
	Name:   "lachesis-update",
	Usage:  "Computes pseudo transition that transits the last world state of Lachesis to the world state of Opera in block in 4,564,026",
	Flags: []cli.Flag{
		&utils.ChainIDFlag,
		&utils.DeletionDbFlag,
		&substate.SubstateDbFlag,
		&substate.UpdateDbFlag,
		&substate.WorkersFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The lachesis-update command requires zero aguments. It compares the initial world state 
the final state of Opera and the final state of Lachesis, then generate a difference set
between the two.`}

func lachesisUpdate(ctx *cli.Context) error {
	// process arguments and flags
	if ctx.Args().Len() != 0 {
		return fmt.Errorf("lachesis-update command requires exactly 0 arguments")
	}
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}
	log := logger.NewLogger(cfg.LogLevel, "Lachesis Update")

	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDB()
	defer substate.CloseSubstateDB()

	// load initial opera initial state in updateset format
	log.Notice("Load Opera initial world state")
	opera, err := utildb.LoadOperaWorldState(cfg.UpdateDb)
	if err != nil {
		return err
	}

	log.Notice("Generate Lachesis world state")
	lachesis, err := utildb.CreateLachesisWorldState(cfg)
	if err != nil {
		return err
	}

	//check if transition tx exists
	lastTx, _ := substate.GetLastSubstate()
	lachesisLastBlock := utils.FirstOperaBlock - 1
	untrackedState := make(substate.SubstateAlloc)

	if lastTx.Env.Number < lachesisLastBlock {
		// update untracked changes
		log.Notice("Calculate difference set")
		untrackedState = opera.Diff(lachesis)
		// create a transition transaction
		lastTx.Env.Number = lachesisLastBlock
		transitionTx := substate.NewSubstate(
			make(substate.SubstateAlloc),
			untrackedState,
			lastTx.Env,
			lastTx.Message,
			lastTx.Result)
		// replace lachesis storage with zeros
		if err := utildb.FixSfcContract(lachesis, transitionTx); err != nil {
			return err
		}

		// write to db
		log.Noticef("Write a transition tx to Block %v Tx %v with %v accounts",
			lachesisLastBlock,
			utils.PseudoTx,
			len(untrackedState))
		substate.PutSubstate(lachesisLastBlock, utils.PseudoTx, transitionTx)
	} else {
		log.Warningf("Transition tx has already been produced. Skip writing")
	}
	return nil
}
