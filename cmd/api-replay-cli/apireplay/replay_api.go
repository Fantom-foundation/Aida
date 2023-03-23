package apireplay

import (
	"context"
	"log"
	"path/filepath"
	"sync"

	"github.com/Fantom-foundation/Aida/cmd/api-replay-cli/flags"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

func ReplayAPI(ctx *cli.Context) error {
	reader, err := iterator.NewFileReader(context.Background(), ctx.String(flags.APIRecordingSrcFileFlag.Name))

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		log.Fatalf("cannot create cfg. Err: %v", err)
	}

	// create StateDB
	dbInfo, err := utils.ReadStateDbInfo(filepath.Join(cfg.StateDbSrcDir, utils.DbInfoName))
	if err != nil {
		log.Fatalf("cannot read db info. Err: %v", err)
	}

	db, err := utils.MakeStateDB(cfg.StateDbSrcDir, cfg, dbInfo.RootHash, true)
	if err != nil {
		log.Fatalf("cannot mate state db. Err: %v", err)
	}

	wg := new(sync.WaitGroup)

	// create sender and start it
	sender := newReplayExecutor(db, reader, cfg, newLogger(ctx), wg)
	sender.Start()

	// create comparator and start it
	comparator := newComparator(sender.output, newLogger(ctx), wg)
	comparator.Start()

	wg.Wait()

	return nil
}
