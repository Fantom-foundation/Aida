package runvm

import (
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

//TODO make flags private
// RunVMApp data structure
var RunVMCommand = cli.Command{
	Action:    RunVM,
	Name:      "Aida Storage Run VM Manager",
	HelpName:  "runvm",
	Usage:     "run VM on the world-state",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		// AidaDb
		&AidaDbFlag,
		&substate.SubstateDbFlag,
		&DeletionDbFlag,
		&UpdateDbFlag,

		// StateDb
		&CarmenSchemaFlag,
		&StateDbImplementationFlag,
		&StateDbVariantFlag,
		&StateDbSrcFlag,
		&DbTmpFlag,
		&StateDbLoggingFlag,

		// ArchiveDb
		&ArchiveModeFlag,
		&ArchiveVariantFlag,

		// ShadowDb
		&ShadowDb,
		&ShadowDbImplementationFlag,
		&ShadowDbVariantFlag,

		// VM
		&VmImplementation,

		// Profiling
		&CpuProfileFlag,
		&MemoryBreakdownFlag,
		&MemoryProfileFlag,
		&RandomSeedFlag,
		&PrimeThresholdFlag,
		&ProfileFlag,

		// Priming
		&RandomizePrimingFlag,
		&SkipPrimingFlag,
		&UpdateBufferSizeFlag,

		// Utils
		&substate.WorkersFlag,
		&ChainIDFlag,
		&ContinueOnFailureFlag,
		&QuietFlag,
		&SyncPeriodLengthFlag,
		&KeepDbFlag,
		&MaxNumTransactionsFlag,
		&ValidateTxStateFlag,
		&ValidateWorldStateFlag,
		&ValidateFlag,
		&LogLevelFlag,

	},
	Description: `
The run-vm command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}




