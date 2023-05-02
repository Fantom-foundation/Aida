package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/runvm-cli/runvm"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RunVMApp data structure
var RunVMApp = cli.App{
	Action:    runvm.RunVM,
	Name:      "Aida Storage Run VM Manager",
	HelpName:  "runvm",
	Usage:     "run VM on the world-state",
	Copyright: "(c) 2022 Fantom Foundation",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SubstateDirFlag,
		&utils.ArchiveModeFlag,
		&utils.ArchiveVariantFlag,
		&utils.CarmenSchemaFlag,
		&utils.ChainIDFlag,
		&utils.ContinueOnFailureFlag,
		&utils.CpuProfileFlag,
		&utils.DeletionDbFlag,
		&utils.QuietFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.KeepDbFlag,
		&utils.MaxNumTransactionsFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,
		&utils.PrimeSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&utils.UpdateDbFlag,
		&utils.ValidateTxStateFlag,
		&utils.ValidateWorldStateFlag,
		&utils.ValidateFlag,
		&utils.VmImplementation,
		&utils.AidaDbFlag,
		&utils.LogLevelFlag,
	},
	Description: `
The run-vm command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

// main implements runvm cli.
func main() {
	if err := RunVMApp.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
