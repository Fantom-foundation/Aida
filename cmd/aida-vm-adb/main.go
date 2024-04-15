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

package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RunArchiveApp defines metadata and configuration options the vm-adb executable.
var RunArchiveApp = cli.App{
	Action:    RunVmAdb,
	Name:      "Aida Archive Evaluation Tool",
	HelpName:  "vm-adb",
	Usage:     "run VM on the archive",
	Copyright: "(c) 2023 Fantom Foundation",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	// TODO: derive supported flags from utilized executor extensions (issue #664).
	Flags: []cli.Flag{
		// substate
		&substate.WorkersFlag,

		// utils
		&utils.CpuProfileFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
		&utils.StateDbLoggingFlag,
		&utils.TrackProgressFlag,
		&utils.NoHeartbeatLoggingFlag,
		&utils.ErrorLoggingFlag,

		// StateDb
		&utils.AidaDbFlag,
		&utils.StateDbSrcFlag,
		&utils.ValidateTxStateFlag,

		// ShadowDb
		&utils.ShadowDb,

		// VM
		&utils.VmImplementation,
	},
	Description: "Runs transactions on historic states derived from an archive DB",
}

// main implements vm-sdb cli.
func main() {
	if err := RunArchiveApp.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
