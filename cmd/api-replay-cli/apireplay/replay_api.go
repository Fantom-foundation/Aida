package apireplay

import (
	"context"
	"fmt"
	"path/filepath"

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
		err    error
		fr     *iterator.FileReader
		cfg    *utils.Config
		dbInfo utils.StateDbInfo
		db     state.StateDB
		stats  *operation.ProfileStats
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
	dbInfo, err = utils.ReadStateDbInfo(filepath.Join(cfg.StateDbSrc, utils.DbInfoName))
	if err != nil {
		return err
	}

	db, err = utils.MakeStateDB(cfg.StateDbSrc, cfg, dbInfo.RootHash, true)
	if err != nil {
		return err
	}

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

	if cfg.Profile {
		fmt.Println("=================Statistics=================")
		stats.PrintProfiling(utils.NewLogger(cfg.LogLevel, "Profile"))
		fmt.Println("============================================")
	}

	return err
}
