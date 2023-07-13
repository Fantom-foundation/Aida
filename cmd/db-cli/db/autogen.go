package db

import (
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// AutoGenCommand generates aida-db patches and handles second opera for event generation
var AutoGenCommand = cli.Command{
	Action: autogen,
	Name:   "autogen",
	Usage:  "autogen generates aida-db periodically",
	Flags: []cli.Flag{
		// TODO minimal epoch length for patch generation
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&utils.DbFlag,
		&utils.GenesisFlag,
		&utils.WorldStateFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.OutputFlag,
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

	stopAtEpoch, err := g.calculatePatchEnd()
	if err != nil {
		return err
	}

	g.log.Noticef("Starting substate generation %d - %d", g.opera.lastEpoch, stopAtEpoch)

	// stop opera to be able to export events
	errCh := startOperaRecording(g.cfg, stopAtEpoch)

	// wait for opera recording response
	err, ok := <-errCh
	if ok && err != nil {
		return err
	}
	g.log.Noticef("Successfully recorded opera: %v substates until: %d", g.cfg.Db, stopAtEpoch)

	err = g.Generate()
	if err != nil {
		return err
	}

	return nil
}
