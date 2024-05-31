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

package db

import (
	"github.com/urfave/cli/v2"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
)

var RunPrimerCmd = cli.Command{
	Action:    RunPrimer,
	Name:      "priming",
	Usage:     "Performs priming of the specified database",
	ArgsUsage: "<blockNum>",
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
		&utils.CustomDbNameFlag,
		&utils.ValidateTxStateFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
		&utils.NoHeartbeatLoggingFlag,
		&utils.TrackProgressFlag,
		&utils.ErrorLoggingFlag,
	},
	Description: `
The util-primer priming command requires one argument: <blockNum>

<blockNum> is the block to which the priming will start.`,
}
