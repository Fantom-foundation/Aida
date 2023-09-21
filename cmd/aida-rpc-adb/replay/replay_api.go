package replay

import (
	"context"
	"fmt"

	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/state/proxy"
	traceCtx "github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/profile"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

func ReplayAPI(ctx *cli.Context) error {
	var (
		err   error
		fr    *iterator.FileReader
		cfg   *utils.Config
		db    state.StateDB
		stats *profile.Stats
	)

	cfg, err = utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	fr, err = iterator.NewFileReader(context.Background(), cfg.APIRecordingSrcFile)
	if err != nil {
		return err
	}

	// there is no need to make temporary copy of stateDb because it isn't modified by this command
	cfg.CopySrcDb = false

	// create StateDB
	db, _, err = utils.PrepareStateDB(cfg)
	if err != nil {
		return fmt.Errorf("cannot prepare StateDb; %v", err)
	}

	// Enable tracing if debug flag is set
	if cfg.Trace {
		rCtx, err := traceCtx.NewRecord(cfg.TraceFile, cfg.First)
		if err != nil {
			return err
		}
		defer rCtx.Close()
		db = proxy.NewRecorderProxy(db, rCtx)
	}

	if cfg.Profile {
		db, stats = proxy.NewProfilerProxy(db, cfg.ProfileFile, cfg.LogLevel)
	}

	err = utils.StartCPUProfile(cfg)
	if err != nil {
		return err
	}

	err = utils.StartMemoryProfile(cfg)
	if err != nil {
		return err
	}
	// closing gracefully both Substate and StateDB is necessary
	defer func() error {
		err = db.Close()
		if err != nil {
			return fmt.Errorf("cannot close db; %v", err)
		}
		return nil
	}()

	// start the rpc replay
	r, err := newController(ctx, cfg, db, fr, stats)
	if err != nil {
		return err
	}

	r.Start()

	r.Wait()

	return nil
}
