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

			// ShadowDB
			&utils.ShadowDb,

			// StateDB
			&utils.StateDbSrcFlag,
			&utils.StateDbLoggingFlag,
			&utils.CarmenSchemaFlag,

			// ArchiveDB
			&utils.ArchiveModeFlag,
			&utils.ArchiveVariantFlag,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
