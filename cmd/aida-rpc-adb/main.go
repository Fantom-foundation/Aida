package main

import (
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/aida-rpc-adb/replay"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Action: replay.RunRPCAdb,
		Name:   "Replay-API",
		Usage: "Sends real API requests recorded on rpcapi.fantom.network to StateDB then compares recorded" +
			"result with result returned by DB.",
		Copyright: "(c) 2023 Fantom Foundation",
		Flags: []cli.Flag{
			&utils.RPCRecordingFileFlag,
			&substate.WorkersFlag,
			&utils.LogFileFlag,

			// VM
			&utils.VmImplementation,

			// Config
			&logger.LogLevelFlag,
			&utils.ChainIDFlag,
			&utils.ContinueOnFailureFlag,

			// ShadowDB
			&utils.ShadowDb,

			// StateDB
			&utils.StateDbSrcFlag,
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
			&utils.ProfileFileFlag,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
