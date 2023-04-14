package apireplay

import (
	"context"
	"log"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/cmd/api-replay-cli/flags"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func ReplayAPI(ctx *cli.Context) error {
	iter, err := iterator.NewFileReader(context.Background(), ctx.String(flags.APIRecordingSrcFileFlag.Name))
	if err != nil {
		log.Fatalf("cannot start iter; err: %v", err)
	}

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		log.Fatalf("cannot create cfg; err: %v", err)
	}

	// create StateDB
	dbInfo, err := utils.ReadStateDbInfo(filepath.Join(cfg.StateDbSrcDir, utils.DbInfoName))
	if err != nil {
		log.Fatalf("cannot read db info; err: %v", err)
	}

	db, err := utils.MakeStateDB(cfg.StateDbSrcDir, cfg, dbInfo.RootHash, true)
	if err != nil {
		log.Fatalf("cannot make state db; err: %v", err)
	}

	substate.SetSubstateDirectory(cfg.SubstateDBDir)
	substate.OpenSubstateDBReadOnly()
	//defer substate.CloseSubstateDB()

	r := newController(ctx, cfg, db, iter)
	r.Start()

	r.Wait()
	return nil
}
