package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
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
		&utils.WorkersFlag,

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
