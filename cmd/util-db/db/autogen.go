package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
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
		&utils.OperaDbFlag,
		&utils.GenesisFlag,
		&utils.DbTmpFlag,
		&utils.OperaBinaryFlag,
		&utils.OutputFlag,
		&utils.TargetEpochFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.WorldStateFlag,
		&utils.WorkersFlag,
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

	locked, err := utildb.GetLock(cfg)
	if err != nil {
		return err
	}
	if locked != "" {
		return fmt.Errorf("GENERATION BLOCKED: autogen failed in last run; %v", locked)
	}

	var g *utildb.Generator
	var ok bool
	g, ok, err = utildb.PrepareAutogen(ctx, cfg)
	if err != nil {
		return fmt.Errorf("cannot start autogen; %v", err)
	}
	if !ok {
		g.Log.Warningf("supplied targetEpoch %d is already reached; latest generated epoch %d", g.TargetEpoch, g.Opera.FirstEpoch-1)
		return nil
	}

	err = utildb.AutogenRun(cfg, g)
	if err != nil {
		errLock := utildb.SetLock(cfg, err.Error())
		if errLock != nil {
			return fmt.Errorf("%v; %v", errLock, err)
		}
	}
	return err
}
