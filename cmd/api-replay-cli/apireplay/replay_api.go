package apireplay

import (
	"context"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/cmd/api-replay-cli/flags"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func ReplayAPI(ctx *cli.Context) error {
	var (
		err    error
		fr     *iterator.FileReader
		cfg    *utils.Config
		dbInfo utils.StateDbInfo
		db     state.StateDB
	)

	fr, err = iterator.NewFileReader(context.Background(), ctx.String(flags.APIRecordingSrcFileFlag.Name))
	if err != nil {
		return err
	}

	cfg, err = utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	// create StateDB
	dbInfo, err = utils.ReadStateDbInfo(filepath.Join(cfg.StateDbSrc, utils.DbInfoName))
	if err != nil {
		return err
	}

	db, err = utils.MakeStateDB(cfg.StateDbSrc, cfg, dbInfo.RootHash, true)
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
