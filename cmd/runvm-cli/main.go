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
		// AidaDb
		&utils.AidaDbFlag,
		&substate.SubstateFlag,
		&utils.DeletionDbFlag,
		&utils.UpdateDbFlag,

		// StateDb
		&utils.CarmenSchemaFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,

		// ArchiveDb
		&utils.ArchiveModeFlag,
		&utils.ArchiveVariantFlag,

		// ShadowDb
		&utils.ShadowDb,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,

		// VM
		&utils.VmImplementation,

		// Profiling
		&utils.CpuProfileFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,
		&utils.RandomSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,

		// Priming
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.UpdateBufferSizeFlag,

		// Utils
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.ContinueOnFailureFlag,
		&utils.QuietFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.KeepDbFlag,
		&utils.MaxNumTransactionsFlag,
		&utils.ValidateTxStateFlag,
		&utils.ValidateWorldStateFlag,
		&utils.ValidateFlag,
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
