package apireplay

import (
	"context"
	"log"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer"
	traceCtx "github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/utils"
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

	// Enable tracing if debug flag is set
	if cfg.Trace {
		rCtx := traceCtx.NewRecord(cfg.TraceFile)
		defer rCtx.Close()
		db = tracer.NewProxyRecorder(db, rCtx)
	}

	defer func() {
		err = db.Close()
		if err != nil {
			log.Fatalf("cannot close db; %v", err)
		}
	}()

	// start the replay
	r := newController(ctx, cfg, db, fr)
	r.Start()

	r.Wait()

	return err
}
