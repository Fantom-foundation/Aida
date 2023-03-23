package apireplay

import (
	"context"
	"log"
	"path/filepath"
	"sync"

	"github.com/Fantom-foundation/Aida/cmd/api-replay-cli/flags"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func ReplayAPI(ctx *cli.Context) error {
	reader, err := iterator.NewFileReader(context.Background(), ctx.String(flags.APIRecordingSrcFileFlag.Name))
	if err != nil {
		log.Fatalf("cannot start reader; err: %v", err)
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
		log.Fatalf("cannot mate state db; err: %v", err)
	}

	wg := new(sync.WaitGroup)

	// create executor and comparator and start it
	sender := newReplayExecutor(cfg.First, cfg.Last, db, reader, newLogger(ctx), wg)
	comparator := newComparator(sender.output, newLogger(ctx), wg)

	// start
	sender.Start(ctx.Int(substate.WorkersFlag.Name), cfg)
	comparator.Start()

	wg.Wait()

	return nil
}
