package trace

import (
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// TraceReplayCommand data structure for the replay app
var TraceReplayCommand = cli.Command{
	Action:    ReplayTrace,
	Name:      "replay",
	Usage:     "executes storage trace",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.CarmenSchemaFlag,
		&utils.ChainIDFlag,
		&utils.CpuProfileFlag,
		&utils.QuietFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.KeepDbFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,
		&utils.RandomSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.ProfileFileFlag,
		&utils.ProfileIntervalFlag,
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.VmImplementation,
		&utils.DbTmpFlag,
		&utils.UpdateBufferSizeFlag,
		&utils.StateDbLoggingFlag,
		&utils.ShadowDb,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&substate.WorkersFlag,
		&utils.TraceFileFlag,
		&utils.TraceDirectoryFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.ValidateFlag,
		&utils.ValidateWorldStateFlag,
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The trace replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// TraceReplaySubstateCommand data structure for the replay-substate app
var TraceReplaySubstateCommand = cli.Command{
	Action:    ReplaySubstate,
	Name:      "replay-substate",
	Usage:     "executes storage trace using substates",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.ChainIDFlag,
		&utils.CpuProfileFlag,
		&utils.QuietFlag,
		&utils.RandomizePrimingFlag,
		&utils.RandomSeedFlag,
		&utils.PrimeThresholdFlag,
		&utils.ProfileFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbLoggingFlag,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,
		&utils.SyncPeriodLengthFlag,
		&substate.WorkersFlag,
		&utils.TraceFileFlag,
		&utils.TraceDirectoryFlag,
		&utils.TraceDebugFlag,
		&utils.DebugFromFlag,
		&utils.ValidateFlag,
		&utils.ValidateWorldStateFlag,
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The trace replay-substate command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}
