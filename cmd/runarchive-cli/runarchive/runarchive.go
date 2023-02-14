package runarchive

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/Fantom-foundation/Aida/cmd/runvm-cli/runvm"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// RunArchive implements the command evaluating historic transactions on an archive.
func RunArchive(ctx *cli.Context) error {
	var (
		err         error
		start       time.Time
		sec         float64
		lastSec     float64
		txCount     int
		lastTxCount int
	)

	// process general arguments
	cfg, argErr := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if argErr != nil {
		return argErr
	}

	// start CPU profiling if requested
	if profileFileName := ctx.String(utils.CpuProfileFlag.Name); profileFileName != "" {
		f, err := os.Create(profileFileName)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %s", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %s", err)
		}
		defer pprof.StopCPUProfile()
	}

	// open the archive
	db, err := openStateDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	// open substate DB
	substate.SetSubstateFlags(ctx)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	log.Printf("Running transactions on archive ...\n")
	iter := substate.NewSubstateIterator(cfg.First, cfg.Workers)
	defer iter.Release()

	if cfg.EnableProgress {
		start = time.Now()
		lastSec = time.Since(start).Seconds()
	}

	var currentBlock uint64 = 0
	var state state.StateDB
	for iter.Next() {
		// Transactions within the same block need to be processed in order,
		// using the same state instance since effects of transactions earlier
		// in a block need to be visible for transactions later in the block.
		tx := iter.Value()

		// Fetch the next block as needed.
		if tx.Block != currentBlock {
			if tx.Block > cfg.Last {
				break
			}
			// For running transactions in block X we need the snapshot of X-1
			if state, err = db.GetArchiveState(tx.Block - 1); err != nil {
				return err
			}
			state.BeginBlock(tx.Block)
			currentBlock = tx.Block
		}

		//log.Printf("\tRunning %d/%d ..", tx.Block, tx.Transaction)
		state.BeginTransaction(uint32(tx.Transaction))
		if _, err = runvm.RunVMTask(state, cfg, tx.Block, tx.Transaction, tx.Substate); err != nil {
			return err
		}
		state.EndTransaction()

		if cfg.EnableProgress {
			txCount++

			// report progress
			sec = time.Since(start).Seconds()

			// Report progress on a regular time interval (wall time).
			if sec-lastSec >= 15 {
				txRate := float64(txCount-lastTxCount) / (sec - lastSec)
				log.Printf("Elapsed time: %.0f s, at block %v (~ %.1f Tx/s)\n", sec, tx.Block, txRate)
				lastSec = sec
				lastTxCount = txCount
			}
		}
	}

	// print progress summary
	if cfg.EnableProgress {
		runTime := time.Since(start).Seconds()
		log.Printf("Total elapsed time: %.3f s, processed %v blocks, %v transactions (~ %.1f Tx/s)\n", runTime, cfg.Last-cfg.First+1, txCount, float64(txCount)/(runTime))
	}

	return err
}

func openStateDB(cfg *utils.Config) (state.StateDB, error) {
	var err error

	if cfg.StateDbSrcDir == "" {
		return nil, fmt.Errorf("missing --db-src-dir parameter")
	}

	// check if statedb_info.json files exist
	dbInfoFile := filepath.Join(cfg.StateDbSrcDir, utils.DbInfoName)
	if _, err = os.Stat(dbInfoFile); err != nil {
		return nil, fmt.Errorf("%s does not appear to contain a state DB", cfg.StateDbSrcDir)
	}

	dbinfo, ferr := utils.ReadStateDbInfo(dbInfoFile)
	if ferr != nil {
		return nil, fmt.Errorf("failed to read %v. %v", dbInfoFile, ferr)
	}
	if dbinfo.Impl != cfg.DbImpl {
		err = fmt.Errorf("mismatch DB implementation.\n\thave %v\n\twant %v", dbinfo.Impl, cfg.DbImpl)
	} else if dbinfo.Variant != cfg.DbVariant {
		err = fmt.Errorf("mismatch DB variant.\n\thave %v\n\twant %v", dbinfo.Variant, cfg.DbVariant)
	} else if dbinfo.Block < cfg.Last {
		err = fmt.Errorf("the state DB does not cover the targeted block range.\n\thave %v\n\twant %v", dbinfo.Block, cfg.Last)
	} else if !dbinfo.ArchiveMode {
		err = fmt.Errorf("the targeted state DB does not include an archive")
	}
	if err != nil {
		return nil, err
	}

	cfg.ArchiveMode = true
	return utils.MakeStateDB(cfg.StateDbSrcDir, cfg, dbinfo.RootHash, true)
}
