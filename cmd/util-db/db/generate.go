// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
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

// GenerateCommand data structure for the replay app
var GenerateCommand = cli.Command{
	Action: generate,
	Name:   "generate",
	Usage:  "generates full aida-db from substatedb - generates deletiondb and updatesetdb, merges them into aida-db and then creates a patch",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&utils.OperaDbFlag,
		&utils.OperaBinaryFlag,
		&utils.OutputFlag,
		&utils.DbTmpFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.SkipStateHashScrappingFlag,
		&substate.WorkersFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The db generate command requires 4 arguments:
<firstBlock> <lastBlock> <firstEpoch> <lastEpoch>
This command is designed for manual generation of deletion, updateset and patch just from substates in aidadb.
`,
}

// generate AidaDb
func generate(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return fmt.Errorf("cannot create config %v", err)
	}

	g, err := utildb.NewGenerator(ctx, cfg)
	if err != nil {
		return err
	}

	err = g.PrepareManualGenerate(ctx, cfg)
	if err != nil {
		return fmt.Errorf("prepareManualGenerate: %v; make sure you have correct substate range already recorded in aidadb", err)
	}

	if err = g.Generate(); err != nil {
		return err
	}

	utildb.MustCloseDB(g.AidaDb)

	return utildb.PrintMetadata(g.Cfg.AidaDb)
}
