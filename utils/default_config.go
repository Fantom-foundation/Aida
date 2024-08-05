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

package utils

import (
	"github.com/Fantom-foundation/Aida/cmd/util-db/flags"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/urfave/cli/v2"
)

// createConfigFromFlags returns Config instance with user specified values or the default ones
func createConfigFromFlags(ctx *cli.Context) *Config {
	cfg := &Config{
		AppName:     ctx.App.HelpName,
		CommandName: ctx.Command.Name,

		AidaDb:                   getFlagValue(ctx, AidaDbFlag).(string),
		ArchiveMaxQueryAge:       getFlagValue(ctx, ArchiveMaxQueryAgeFlag).(int),
		ArchiveMode:              getFlagValue(ctx, ArchiveModeFlag).(bool),
		ArchiveQueryRate:         getFlagValue(ctx, ArchiveQueryRateFlag).(int),
		ArchiveVariant:           getFlagValue(ctx, ArchiveVariantFlag).(string),
		BalanceRange:             getFlagValue(ctx, BalanceRangeFlag).(int64),
		BasicBlockProfiling:      getFlagValue(ctx, BasicBlockProfilingFlag).(bool),
		BlockLength:              getFlagValue(ctx, BlockLengthFlag).(uint64),
		CPUProfile:               getFlagValue(ctx, CpuProfileFlag).(string),
		CPUProfilePerInterval:    getFlagValue(ctx, CpuProfilePerIntervalFlag).(bool),
		Cache:                    getFlagValue(ctx, CacheFlag).(int),
		CarmenSchema:             getFlagValue(ctx, CarmenSchemaFlag).(int),
		CarmenCheckpointInterval: getFlagValue(ctx, CarmenCheckpointInterval).(int),
		CarmenCheckpointPeriod:   getFlagValue(ctx, CarmenCheckpointPeriod).(int),
		ChainID:                  ChainID(getFlagValue(ctx, ChainIDFlag).(int)),
		ChannelBufferSize:        getFlagValue(ctx, ChannelBufferSizeFlag).(int),
		CompactDb:                getFlagValue(ctx, CompactDbFlag).(bool),
		ContinueOnFailure:        getFlagValue(ctx, ContinueOnFailureFlag).(bool),
		ContractNumber:           getFlagValue(ctx, ContractNumberFlag).(int64),
		CustomDbName:             getFlagValue(ctx, CustomDbNameFlag).(string),
		DbComponent:              getFlagValue(ctx, DbComponentFlag).(string),
		DbImpl:                   getFlagValue(ctx, StateDbImplementationFlag).(string),
		DbLogging:                getFlagValue(ctx, StateDbLoggingFlag).(string),
		DbTmp:                    getFlagValue(ctx, DbTmpFlag).(string),
		DbVariant:                getFlagValue(ctx, StateDbVariantFlag).(string),
		Debug:                    getFlagValue(ctx, TraceDebugFlag).(bool),
		DebugFrom:                getFlagValue(ctx, DebugFromFlag).(uint64),
		DeleteSourceDbs:          getFlagValue(ctx, DeleteSourceDbsFlag).(bool),
		DeletionDb:               getFlagValue(ctx, DeletionDbFlag).(string),
		DiagnosticServer:         getFlagValue(ctx, DiagnosticServerFlag).(int64),
		ErrorLogging:             getFlagValue(ctx, ErrorLoggingFlag).(string),
		Forks:                    getFlagValue(ctx, ForksFlag).([]string),
		Genesis:                  getFlagValue(ctx, GenesisFlag).(string),
		EthTestType:              EthTestType(getFlagValue(ctx, EthTestTypeFlag).(int)),
		IncludeStorage:           getFlagValue(ctx, IncludeStorageFlag).(bool),
		KeepDb:                   getFlagValue(ctx, KeepDbFlag).(bool),
		KeysNumber:               getFlagValue(ctx, KeysNumberFlag).(int64),
		LogLevel:                 getFlagValue(ctx, logger.LogLevelFlag).(string),
		MaxNumErrors:             getFlagValue(ctx, MaxNumErrorsFlag).(int),
		MaxNumTransactions:       getFlagValue(ctx, MaxNumTransactionsFlag).(int),
		MemoryBreakdown:          getFlagValue(ctx, MemoryBreakdownFlag).(bool),
		MemoryProfile:            getFlagValue(ctx, MemoryProfileFlag).(string),
		MicroProfiling:           getFlagValue(ctx, MicroProfilingFlag).(bool),
		NoHeartbeatLogging:       getFlagValue(ctx, NoHeartbeatLoggingFlag).(bool),
		NonceRange:               getFlagValue(ctx, NonceRangeFlag).(int),
		OnlySuccessful:           getFlagValue(ctx, OnlySuccessfulFlag).(bool),
		OperaBinary:              getFlagValue(ctx, OperaBinaryFlag).(string),
		OperaDb:                  getFlagValue(ctx, OperaDbFlag).(string),
		Output:                   getFlagValue(ctx, OutputFlag).(string),
		OverwriteRunId:           getFlagValue(ctx, OverwriteRunIdFlag).(string),
		PrimeRandom:              getFlagValue(ctx, RandomizePrimingFlag).(bool),
		PrimeThreshold:           getFlagValue(ctx, PrimeThresholdFlag).(int),
		Profile:                  getFlagValue(ctx, ProfileFlag).(bool),
		ProfileBlocks:            getFlagValue(ctx, ProfileBlocksFlag).(bool),
		ProfileDB:                getFlagValue(ctx, ProfileDBFlag).(string),
		ProfileDepth:             getFlagValue(ctx, ProfileDepthFlag).(int),
		ProfileEVMCall:           getFlagValue(ctx, ProfileEVMCallFlag).(bool),
		ProfileFile:              getFlagValue(ctx, ProfileFileFlag).(string),
		ProfileInterval:          getFlagValue(ctx, ProfileIntervalFlag).(uint64),
		ProfileSqlite3:           getFlagValue(ctx, ProfileSqlite3Flag).(string),
		ProfilingDbName:          getFlagValue(ctx, ProfilingDbNameFlag).(string),
		RandomSeed:               getFlagValue(ctx, RandomSeedFlag).(int64),
		RegisterRun:              getFlagValue(ctx, RegisterRunFlag).(string),
		RpcRecordingPath:         getFlagValue(ctx, RpcRecordingFileFlag).(string),
		ShadowDb:                 getFlagValue(ctx, ShadowDb).(bool),
		ShadowImpl:               getFlagValue(ctx, ShadowDbImplementationFlag).(string),
		ShadowVariant:            getFlagValue(ctx, ShadowDbVariantFlag).(string),
		SkipMetadata:             getFlagValue(ctx, flags.SkipMetadata).(bool),
		SkipPriming:              getFlagValue(ctx, SkipPrimingFlag).(bool),
		SkipStateHashScrapping:   getFlagValue(ctx, SkipStateHashScrappingFlag).(bool),
		SnapshotDepth:            getFlagValue(ctx, SnapshotDepthFlag).(int),
		StateDbSrc:               getFlagValue(ctx, StateDbSrcFlag).(string),
		StateDbSrcDirectAccess:   getFlagValue(ctx, StateDbSrcOverwriteFlag).(bool),
		StateDbSrcReadOnly:       false,
		// TODO re-enable equality check once supported in Carmen
		StateValidationMode: SubsetCheck,
		SubstateDb:          getFlagValue(ctx, AidaDbFlag).(string),
		SyncPeriodLength:    getFlagValue(ctx, SyncPeriodLengthFlag).(uint64),
		TargetDb:            getFlagValue(ctx, TargetDbFlag).(string),
		TargetEpoch:         getFlagValue(ctx, TargetEpochFlag).(uint64),
		Trace:               getFlagValue(ctx, TraceFlag).(bool),
		TraceDirectory:      getFlagValue(ctx, TraceDirectoryFlag).(string),
		TraceFile:           getFlagValue(ctx, TraceFileFlag).(string),
		TrackProgress:       getFlagValue(ctx, TrackProgressFlag).(bool),
		TransactionLength:   getFlagValue(ctx, TransactionLengthFlag).(uint64),
		UpdateBufferSize:    getFlagValue(ctx, UpdateBufferSizeFlag).(uint64),
		UpdateDb:            getFlagValue(ctx, UpdateDbFlag).(string),
		UpdateOnFailure:     getFlagValue(ctx, UpdateOnFailure).(bool),
		UpdateType:          getFlagValue(ctx, UpdateTypeFlag).(string),
		Validate:            getFlagValue(ctx, ValidateFlag).(bool),
		ValidateStateHashes: getFlagValue(ctx, ValidateStateHashesFlag).(bool),
		ValidateTxState:     getFlagValue(ctx, ValidateTxStateFlag).(bool),
		ValuesNumber:        getFlagValue(ctx, ValuesNumberFlag).(int64),
		VmImpl:              getFlagValue(ctx, VmImplementation).(string),
		Workers:             getFlagValue(ctx, WorkersFlag).(int),
		TxGeneratorType:     getFlagValue(ctx, TxGeneratorTypeFlag).([]string),
	}

	return cfg
}

// getFlagValue returns value specified by user if flag is present in cli context, otherwise return default flag value
func getFlagValue(ctx *cli.Context, flag interface{}) interface{} {
	cmdFlags := ctx.Command.Flags
	for _, cmdFlag := range cmdFlags {
		switch f := flag.(type) {
		case cli.IntFlag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.Int(f.Name)
			}

		case cli.Uint64Flag:
			if cmdFlag.Names()[0] == UpdateBufferSizeFlag.Name {
				return ctx.Uint64(f.Name) * 1_000_000
			} else if cmdFlag.Names()[0] == f.Name {
				return ctx.Uint64(f.Name)
			}

		case cli.Int64Flag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.Int64(f.Name)
			}

		case cli.StringFlag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.String(f.Name)
			}

		case cli.PathFlag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.Path(f.Name)
			}

		case cli.BoolFlag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.Bool(f.Name)
			}
		case cli.StringSliceFlag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.StringSlice(f.Name)
			}
		}
	}

	// If flag not found, return the default value of the flag
	switch f := flag.(type) {
	case cli.IntFlag:
		return f.Value
	case cli.Uint64Flag:
		return f.Value
	case cli.Int64Flag:
		return f.Value
	case cli.StringFlag:
		return f.Value
	case cli.PathFlag:
		return f.Value
	case cli.BoolFlag:
		return f.Value
	case cli.StringSliceFlag:
		if f.Value == nil {
			return []string{}
		}
		return f.Value.Value()
	}

	return nil
}
