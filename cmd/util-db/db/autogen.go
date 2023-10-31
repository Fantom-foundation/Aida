package db

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// AutoGenCommand generates aida-db patches and handles second opera for event generation
var AutoGenCommand = cli.Command{
	Action: autogen,
	Name:   "autogen",
	Usage:  "autogen generates aida-db periodically",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&utils.DbFlag,
		&utils.GenesisFlag,
		&utils.DbTmpFlag,
		&utils.OperaBinaryFlag,
		&utils.OutputFlag,
		&utils.TargetEpochFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.WorldStateFlag,
		&substate.WorkersFlag,
		&logger.LogLevelFlag,
	},
	Description: `
AutoGen generates aida-db patches and handles second opera for event generation. Generates event file, which is supplied into doGenerations to create aida-db patch.
`,
}

// autogen command is used to record/update aida-db periodically
func autogen(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	locked, err := getLock(cfg)
	if err != nil {
		return err
	}
	if locked != "" {
		return fmt.Errorf("GENERATION BLOCKED: autogen failed in last run; %v", locked)
	}

	var g *generator
	g, err = prepareAutogen(ctx, cfg)
	if err != nil {
		return fmt.Errorf("cannot start autogen; %v", err)
	}

	err = autogenRun(cfg, g)
	if err != nil {
		errLock := setLock(cfg, err.Error())
		if errLock != nil {
			return fmt.Errorf("%v; %v", errLock, err)
		}
	}
	return err
}

// prepareAutogen initializes a generator object, opera binary and adjust target range
func prepareAutogen(ctx *cli.Context, cfg *utils.Config) (*generator, error) {
	g, err := newGenerator(ctx, cfg)
	if err != nil {
		return nil, err
	}

	err = g.opera.init()
	if err != nil {
		return nil, err
	}

	// remove worldstate directory if it was created
	defer func(log *logging.Logger) {
		if cfg.WorldStateDb != "" {
			err = os.RemoveAll(cfg.WorldStateDb)
			if err != nil {
				log.Criticalf("can't remove temporary folder: %v; %v", cfg.WorldStateDb, err)
			}
		}
	}(g.log)

	// user specified targetEpoch
	if cfg.TargetEpoch > 0 {
		g.targetEpoch = cfg.TargetEpoch
	} else {
		err = g.calculatePatchEnd()
		if err != nil {
			return nil, err
		}
	}

	MustCloseDB(g.aidaDb)

	// start epoch is last epoch + 1
	if g.opera.firstEpoch > g.targetEpoch {
		return nil, fmt.Errorf("supplied targetEpoch %d is already reached; latest generated epoch %d", g.targetEpoch, g.opera.firstEpoch-1)
	}
	return g, nil
}

// setLock creates lockfile in case of error while generating
func setLock(cfg *utils.Config, message string) error {
	lockFile := cfg.AidaDb + ".autogen.lock"

	// Write the string to the file
	err := os.WriteFile(lockFile, []byte(message), 0655)
	if err != nil {
		return fmt.Errorf("error writing to lock file %v; %v", lockFile, err)
	} else {
		return nil
	}
}

// getLock checks existence and contents of lockfile
func getLock(cfg *utils.Config) (string, error) {
	lockFile := cfg.AidaDb + ".autogen.lock"

	// Read lockfile contents
	content, err := os.ReadFile(lockFile)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("error reading from file; %v", err)
	}

	return string(content), nil
}

// autogenRun is used to record/update aida-db
func autogenRun(cfg *utils.Config, g *generator) error {
	g.log.Noticef("Starting substate generation %d - %d", g.opera.firstEpoch, g.targetEpoch)

	start := time.Now()
	// stop opera to be able to export events
	errCh := startOperaRecording(g.cfg, g.targetEpoch)

	// wait for opera recording response
	err, ok := <-errCh
	if ok && err != nil {
		return err
	}
	g.log.Noticef("Recording (%v) for epoch range %d - %d finished. It took: %v", g.cfg.Db, g.opera.firstEpoch, g.targetEpoch, time.Since(start).Round(1*time.Second))
	g.log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))

	// reopen aida-db
	g.aidaDb, err = rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot create new db; %v", err)
	}
	substate.SetSubstateDbBackend(g.aidaDb)

	err = g.opera.getOperaBlockAndEpoch(false)
	if err != nil {
		return err
	}

	return g.Generate()
}
