package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/runarchive-cli/runarchive"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// RunArchiveApp defines metadata and configuration options the runarchive executable.
var RunArchiveApp = cli.App{
	Action:    runarchive.RunArchive,
	Name:      "Aida Archive Evaluation Tool",
	HelpName:  "runarchive",
	Usage:     "run VM on the archive",
	Copyright: "(c) 2023 Fantom Foundation",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&utils.CpuProfileFlag,
		&utils.ChainIDFlag,
		&utils.StateDbSrcDirFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.ValidateTxStateFlag,
		&utils.VmImplementation,
	},
	Description: "Runs transactions on historic states derived from an archive DB",
}

// main implements runvm cli.
func main() {
	if err := RunArchiveApp.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
