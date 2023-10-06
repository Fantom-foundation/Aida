package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// RunVMApp data structure
var RunVMApp = cli.App{
	Action:    RunVmSdb,
	Name:      "Aida Storage Run VM Manager",
	HelpName:  "vm-sdb",
	Usage:     "run VM on the world-state",
	Copyright: "(c) 2023 Fantom Foundation",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	// TODO: derive supported flags from utilized executor extensions (issue #664).
	Flags: []cli.Flag{
		// AidaDb
		&utils.AidaDbFlag,

		// StateDb
		&utils.CarmenSchemaFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,
		&utils.StateRootHashesFlag,

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
		&utils.CpuProfilePerIntervalFlag,
		&utils.DiagnosticServerFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,
		&utils.RandomSeedFlag,
		&utils.PrimeThresholdFlag,
		//&utils.ProfileFlag,
		//&utils.ProfileFileFlag,
		//&utils.ProfileIntervalFlag,

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
		//&utils.MaxNumTransactionsFlag,
		&utils.ValidateTxStateFlag,
		//&utils.ValidateWorldStateFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
		&utils.NoHeartbeatLoggingFlag,
		&utils.TrackProgressFlag,
	},
	Description: `
The aida-vm-sdb command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

// main implements vm-sdb cli.
func main() {
	if err := RunVMApp.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
