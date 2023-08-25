package vm

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/tx_processor"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// record-replay: vm replay command
var ReplayCommand = cli.Command{
	Action:    Replay,
	Name:      "replay",
	Usage:     "executes full state transitions and check output consistency",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&substate.WorkersFlag,
		&substate.SkipTransferTxsFlag,
		&substate.SkipCallTxsFlag,
		&substate.SkipCreateTxsFlag,
		&utils.ChainIDFlag,
		&utils.ProfileEVMCallFlag,
		&utils.MicroProfilingFlag,
		&utils.BasicBlockProfilingFlag,
		&utils.ProfilingDbNameFlag,
		&utils.ChannelBufferSizeFlag,
		&utils.VmImplementation,
		&utils.OnlySuccessfulFlag,
		&utils.CpuProfileFlag,
		&utils.StateDbImplementationFlag,
		&utils.AidaDbFlag,
		&logger.LogLevelFlag,
	},
	Description: `
The aida-vm replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay transactions.`,
}

func Replay(ctx *cli.Context) error {
	actions := tx_processor.NewExtensionList([]tx_processor.ProcessorExtensions{
		tx_processor.NewMicroProfileExtension(),
		tx_processor.NewBasicProfileExtension(),
	})

	tp, err := tx_processor.NewTxProcessor(ctx, "vm-replay")
	if err != nil {
		return fmt.Errorf("cannot create tx-processor; %v", err)
	}

	return tp.Run(actions)
}
