package main

import (
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/api-replay-cli/apireplay"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Action: apireplay.ReplayAPI,
		Name:   "Replay-API",
		Usage: "Sends real API requests recorded on rpcapi.fantom.network to StateDB then compares recorded" +
			"result with result returned by DB.",
		Copyright: "(c) 2023 Fantom Foundation",
		Flags: []cli.Flag{
			&utils.APIRecordingSrcFileFlag,
			&substate.WorkersFlag,

			// AidaDB
			&utils.AidaDbFlag,

			// VM
			&utils.VmImplementation,

			// Substate
			&substate.SubstateDirFlag,

			// Config
			&utils.LogLevelFlag,
			&utils.ChainIDFlag,
			&utils.ContinueOnFailureFlag,

			// StateDB
			&utils.StateDbImplementationFlag,
			&utils.StateDbVariantFlag,
			&utils.StateDbSrcFlag,
			&utils.DbTmpFlag,
			&utils.StateDbLoggingFlag,

			// Trace
			&utils.TraceFlag,
			&utils.TraceFileFlag,
			&utils.TraceDebugFlag,

			// ArchiveDB
			&utils.ArchiveModeFlag,
			&utils.ArchiveVariantFlag,

			// Performance
			&utils.CpuProfileFlag,
			&utils.MemoryProfileFlag,
			&utils.ProfileFlag,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
