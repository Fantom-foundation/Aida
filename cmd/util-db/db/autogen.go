package db

import (
	"fmt"
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

	g, err := newGenerator(ctx, cfg)
	if err != nil {
		return err
	}

	err = g.opera.init()
	if err != nil {
		return err
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

	err = g.calculatePatchEnd()
	if err != nil {
		return err
	}

	if cfg.TargetEpoch > 0 {
		g.stopAtEpoch = cfg.TargetEpoch
	}

	g.log.Noticef("Starting substate generation %d - %d", g.opera.lastEpoch+1, g.stopAtEpoch)

	MustCloseDB(g.aidaDb)

	start := time.Now()
	// stop opera to be able to export events
	errCh := startOperaRecording(g.cfg, g.stopAtEpoch)

	// wait for opera recording response
	err, ok := <-errCh
	if ok && err != nil {
		return err
	}
	g.log.Noticef("Recording for epoch range %d - %d finished. It took: %v", g.cfg.Db, g.opera.lastEpoch+1, g.stopAtEpoch, time.Since(start).Round(1*time.Second))
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
