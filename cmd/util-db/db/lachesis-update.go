package db

import (
	"context"
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// address of sfc contract in Hex
const sfcAddrHex = "0xFC00FACE00000000000000000000000000000000"

var LachesisUpdateCommand = cli.Command{
	Action: lachesisUpdate,
	Name:   "lachesis-update",
	Usage:  "Computes pseudo transition that transits the last world state of Lachesis to the world state of Opera in block in 4,564,026",
	Flags: []cli.Flag{
		&utils.ChainIDFlag,
		&utils.DeletionDbFlag,
		&substate.SubstateDbFlag,
		&substate.WorkersFlag,
		&utils.WorldStateFlag,
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

	log.Notice("Load Opera initial world state")
	opera, err := loadOperaWorldState(cfg.WorldStateDb)
	if err != nil {
		return err
	}

	log.Notice("Generate Lachesis world state")
	lachesis, err := createLachesisWorldState(cfg)
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
		if err := fixSfcContract(lachesis, transitionTx); err != nil {
			return err
		}

		// write to db
		log.Noticef("Write a transition tx to Block %v Tx %v with %v accounts",
			lachesisLastBlock,
			utils.PseudoTx,
			len(untrackedState))
		substate.PutSubstate(lachesisLastBlock, utils.PseudoTx, transitionTx)
	} else {
		log.Warningf("Transition transaction has already been produced. Skip writing")
	}
	return nil
}

// loadOperaWorldState loads opera initial world state from worldstate-db as SubstateAlloc
func loadOperaWorldState(path string) (substate.SubstateAlloc, error) {
	worldStateDB, err := snapshot.OpenStateDB(path)
	if err != nil {
		return nil, err
	}
	defer snapshot.MustCloseStateDB(worldStateDB)
	opera, err := worldStateDB.ToSubstateAlloc(context.Background())
	return opera, err
}

// createLachesisWorldState creates update-set from block 0 to the last lachesis block
func createLachesisWorldState(cfg *utils.Config) (substate.SubstateAlloc, error) {
	lachesisLastBlock := utils.FirstOperaBlock - 1
	lachesis, _, err := utils.GenerateUpdateSet(0, nil, lachesisLastBlock, cfg)
	if err != nil {
		return nil, err
	}
	// remove deleted accounts
	if err := utils.DeleteDestroyedAccountsFromWorldState(lachesis, cfg, lachesisLastBlock); err != nil {
		return nil, err
	}
	return lachesis, nil
}

// fixSfcContract reset lachesis's storage keys to zeros while keeping opera keys
func fixSfcContract(lachesis substate.SubstateAlloc, transition *substate.Substate) error {
	sfcAddr := common.HexToAddress(sfcAddrHex)
	lachesisSfc, hasLachesisSfc := lachesis[sfcAddr]
	_, hasTransitionSfc := transition.OutputAlloc[sfcAddr]

	if hasLachesisSfc && hasTransitionSfc {
		// insert keys with zero value to the transition substate if
		// the keys doesn't appear on opera
		for key := range lachesisSfc.Storage {
			if _, found := transition.OutputAlloc[sfcAddr].Storage[key]; !found {
				transition.OutputAlloc[sfcAddr].Storage[key] = common.Hash{}
			}
		}
	} else {
		return fmt.Errorf("the SFC contract is not found.")
	}
	return nil
}
