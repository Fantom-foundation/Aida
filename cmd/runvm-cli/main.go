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
		&utils.ChainIDFlag,
		&utils.ContinueOnFailureFlag,
		&utils.CpuProfileFlag,
		&utils.DeletedAccountDirFlag,
		&utils.DisableProgressFlag,
		&utils.EpochLengthFlag,
		&utils.KeepStateDBFlag,
		&utils.MaxNumTransactionsFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemProfileFlag,
		&utils.PrimeSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbTempDirFlag,
		&utils.StateDbLoggingFlag,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&utils.UpdateDBDirFlag,
		&utils.ValidateTxStateFlag,
		&utils.ValidateWorldStateFlag,
		&utils.ValidateFlag,
		&utils.VmImplementation,
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
