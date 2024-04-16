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

package main

import (
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Action: RunRpc,
		Name:   "Replay-RPC",
		Usage: "Sends real API requests recorded on rpcapi.fantom.network to StateDB then compares recorded" +
			"result with result returned by DB.",
		Copyright: "(c) 2023 Fantom Foundation",
		Flags: []cli.Flag{
			&utils.RpcRecordingFileFlag,
			&substate.WorkersFlag,

			// VM
			&utils.VmImplementation,

			// Config
			&logger.LogLevelFlag,
			&utils.ChainIDFlag,
			&utils.ContinueOnFailureFlag,
			&utils.ValidateFlag,
			&utils.NoHeartbeatLoggingFlag,
			&utils.ErrorLoggingFlag,
			&utils.TrackProgressFlag,

			// Register
			&utils.RegisterRunFlag,
			&utils.OverwriteRunIdFlag,

			// ShadowDB
			&utils.ShadowDb,

			// StateDB
			&utils.StateDbSrcFlag,
			&utils.StateDbLoggingFlag,

			// Trace
			&utils.TraceFlag,
			&utils.TraceFileFlag,
			&utils.TraceDebugFlag,

			// Performance
			&utils.CpuProfileFlag,
			&utils.MemoryProfileFlag,
			&utils.ProfileFlag,
			&utils.ProfileFileFlag,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
