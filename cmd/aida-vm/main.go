package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Action:    RunVm,
		Name:      "EVM evaluation tool",
		HelpName:  "aida-vm",
		Copyright: "(c) 2023 Fantom Foundation",
		ArgsUsage: "<blockNumFirst> <blockNumLast>",
		// TODO: derive supported flags from utilized executor extensions.
		Flags: []cli.Flag{
			&substate.WorkersFlag,
			//&substate.SkipTransferTxsFlag,
			//&substate.SkipCallTxsFlag,
			//&substate.SkipCreateTxsFlag,
			&utils.ChainIDFlag,
			//&utils.ProfileEVMCallFlag,
			//&utils.MicroProfilingFlag,
			//&utils.BasicBlockProfilingFlag,
			//&utils.ProfilingDbNameFlag,
			&utils.ChannelBufferSizeFlag,
			&utils.VmImplementation,
			&utils.ValidateTxStateFlag,
			//&utils.OnlySuccessfulFlag,
			&utils.CpuProfileFlag,
			&utils.DiagnosticServerFlag,
			&utils.AidaDbFlag,
			&logger.LogLevelFlag,
			&utils.ErrorLoggingFlag,
			&utils.StateDbImplementationFlag,
			&utils.StateDbLoggingFlag,
			&utils.CacheFlag,
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
