package apireplay

import (
	"context"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func ReplayAPI(ctx *cli.Context) error {
	var (
		err error
		fr  *iterator.FileReader
		cfg *utils.Config
		db  state.StateDB
	)

	cfg, err = utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	fr, err = iterator.NewFileReader(context.Background(), cfg.APIRecordingSrcFile)
	if err != nil {
		return err
	}

	db, _, err = utils.PrepareStateDB(cfg)
	if err != nil {
		return err
	}

	substate.SetSubstateDirectory(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()

	// closing gracefully both Substate and StateDB is necessary
	defer func() {
		err = db.Close()
		substate.CloseSubstateDB()
	}()

	// start the replay
	r := newController(ctx, cfg, db, fr)
	r.Start()

	r.Wait()

	return err
}
