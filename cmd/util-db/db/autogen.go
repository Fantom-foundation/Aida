// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
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
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&utils.OperaDbFlag,
		&utils.GenesisFlag,
		&utils.DbTmpFlag,
		&utils.OperaBinaryFlag,
		&utils.OutputFlag,
		&utils.TargetEpochFlag,
		&utils.UpdateBufferSizeFlag,
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
