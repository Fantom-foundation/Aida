// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
	Name:      "Aida Storage Run VM Manager",
	Copyright: "(c) 2023 Fantom Foundation",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Commands: []*cli.Command{
		&RunSubstateCmd,
		&RunEthTestsCmd,
		&RunTxGeneratorCmd,
	},
	Description: `
The aida-vm-sdb command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

var RunSubstateCmd = cli.Command{
	Action:    RunSubstate,
	Name:      "substate",
	Usage:     "Iterates over substates that are executed into a StateDb",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
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
		&utils.ValidateStateHashesFlag,

		// ArchiveDb
		&utils.ArchiveModeFlag,
		&utils.ArchiveQueryRateFlag,
		&utils.ArchiveMaxQueryAgeFlag,
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
		&utils.ProfileFlag,
		&utils.ProfileDepthFlag,
		&utils.ProfileFileFlag,
		&utils.ProfileSqlite3Flag,
		&utils.ProfileIntervalFlag,
		&utils.ProfileDBFlag,
		&utils.ProfileBlocksFlag,

		// RegisterRun
		&utils.RegisterRunFlag,
		&utils.OverwriteRunIdFlag,

		// Priming
		&utils.RandomizePrimingFlag,
		&utils.SkipPrimingFlag,
		&utils.UpdateBufferSizeFlag,

		// Utils
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.ContinueOnFailureFlag,
		&utils.SyncPeriodLengthFlag,
		&utils.KeepDbFlag,
		&utils.CustomDbNameFlag,
		//&utils.MaxNumTransactionsFlag,
		&utils.ValidateTxStateFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
		&utils.NoHeartbeatLoggingFlag,
		&utils.TrackProgressFlag,
		&utils.ErrorLoggingFlag,
	},
	Description: `
The aida-vm-sdb substate command requires two arguments: <blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and last block of
the inclusive range of blocks.`,
}

var RunTxGeneratorCmd = cli.Command{
	Action: RunTxGenerator,
	Name:   "tx-generator",
	Usage:  "Generates transactions for specified block range and executes them over StateDb",
	Flags: []cli.Flag{
		// TxGenerator specific flags
		&utils.TxGeneratorTypeFlag,

		// StateDb
		&utils.CarmenSchemaFlag,
		&utils.StateDbImplementationFlag,
		&utils.StateDbVariantFlag,
		&utils.StateDbSrcFlag,
		&utils.DbTmpFlag,
		&utils.StateDbLoggingFlag,
		&utils.ValidateStateHashesFlag,

		// ShadowDb
		&utils.ShadowDb,
		&utils.ShadowDbImplementationFlag,
		&utils.ShadowDbVariantFlag,

		// RegisterRun
		&utils.RegisterRunFlag,
		&utils.OverwriteRunIdFlag,

		// VM
		&utils.VmImplementation,

		// Profiling
		&utils.CpuProfileFlag,
		&utils.CpuProfilePerIntervalFlag,
		&utils.DiagnosticServerFlag,
		&utils.MemoryBreakdownFlag,
		&utils.MemoryProfileFlag,

		// Utils
		&substate.WorkersFlag,
		&utils.ChainIDFlag,
		&utils.ContinueOnFailureFlag,
		&utils.KeepDbFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
		&utils.NoHeartbeatLoggingFlag,
		&utils.BlockLengthFlag,
	},
	Description: `
The aida-vm-sdb tx-generator command requires two arguments: <blockNumFirst> <blockNumLast>

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
