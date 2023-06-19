package apireplay

import (
	"context"
	"fmt"

	"github.com/Fantom-foundation/Aida/cmd/runvm-cli/runvm"
	"github.com/Fantom-foundation/Aida/iterator"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/tracer"
	traceCtx "github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func ReplayAPI(ctx *cli.Context) error {
	var (
		err   error
		fr    *iterator.FileReader
		cfg   *utils.Config
		db    state.StateDB
		stats *operation.ProfileStats
	)

	cfg, err = utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	fr, err = iterator.NewFileReader(context.Background(), cfg.APIRecordingSrcFile)
	if err != nil {
		return err
	}

	// create StateDB
	db, _, err = utils.PrepareStateDB(cfg)
	if err != nil {
		return fmt.Errorf("cannot prepare StateDb; %v", err)
	}

	// Enable tracing if debug flag is set
	if cfg.Trace {
		rCtx := traceCtx.NewRecord(cfg.TraceFile)
		defer rCtx.Close()
		db = tracer.NewProxyRecorder(db, rCtx)
	}

	if cfg.Profile {
		db, stats = runvm.NewProxyProfiler(db)
	}

	err = utils.StartCPUProfile(cfg)
	if err != nil {
		return err
	}

	err = utils.StartMemoryProfile(cfg)
	if err != nil {
		return err
	}

	substate.SetSubstateDb(cfg.SubstateDb)
	substate.OpenSubstateDBReadOnly()
	if cfg.APIRecordingVersion == 0 {
		if cfg.SubstateDb == "" {
			return fmt.Errorf("api recording version 0 needs substate, either define it (--substate-db) or use version 1")
		}
		substate.SetSubstateDb(cfg.SubstateDb)
		substate.OpenSubstateDBReadOnly()
	}

	// closing gracefully both Substate and StateDB is necessary
	defer func() {
		err = db.Close()
		substate.CloseSubstateDB()
	}()

	// start the replay
	r := newController(ctx, cfg, db, fr, stats)
	r.Start()

	r.Wait()

	return err
}
