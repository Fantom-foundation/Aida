package utils

import (
	"github.com/Fantom-foundation/Aida/cmd/util-db/flags"
	"github.com/Fantom-foundation/Aida/logger"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// createConfigFromFlags returns Config instance with user specified values or the default ones
func createConfigFromFlags(ctx *cli.Context) *Config {
	cfg := &Config{
		AppName:     ctx.App.HelpName,
		CommandName: ctx.Command.Name,

		APIRecordingSrcFile:   getFlagValue(ctx, APIRecordingSrcFileFlag).(string),
		ArchiveMode:           getFlagValue(ctx, ArchiveModeFlag).(bool),
		ArchiveVariant:        getFlagValue(ctx, ArchiveVariantFlag).(string),
		BlockLength:           getFlagValue(ctx, BlockLengthFlag).(uint64),
		BalanceRange:          getFlagValue(ctx, BalanceRangeFlag).(int64),
		CarmenSchema:          getFlagValue(ctx, CarmenSchemaFlag).(int),
		ChainID:               ChainID(getFlagValue(ctx, ChainIDFlag).(int)),
		Cache:                 getFlagValue(ctx, CacheFlag).(int),
		ContractNumber:        getFlagValue(ctx, ContractNumberFlag).(int64),
		ContinueOnFailure:     getFlagValue(ctx, ContinueOnFailureFlag).(bool),
		SrcDbReadonly:         false,
		CPUProfile:            getFlagValue(ctx, CpuProfileFlag).(string),
		CPUProfilePerInterval: getFlagValue(ctx, CpuProfilePerIntervalFlag).(bool),
		Db:                    getFlagValue(ctx, DbFlag).(string),
		DbTmp:                 getFlagValue(ctx, DbTmpFlag).(string),
		Debug:                 getFlagValue(ctx, TraceDebugFlag).(bool),
		DebugFrom:             getFlagValue(ctx, DebugFromFlag).(uint64),
		Quiet:                 getFlagValue(ctx, QuietFlag).(bool),
		SyncPeriodLength:      getFlagValue(ctx, SyncPeriodLengthFlag).(uint64),
		Genesis:               getFlagValue(ctx, GenesisFlag).(string),
		DbImpl:                getFlagValue(ctx, StateDbImplementationFlag).(string),
		DbVariant:             getFlagValue(ctx, StateDbVariantFlag).(string),
		DbLogging:             getFlagValue(ctx, StateDbLoggingFlag).(bool),
		DeletionDb:            getFlagValue(ctx, DeletionDbFlag).(string),
		DeleteSourceDbs:       getFlagValue(ctx, DeleteSourceDbsFlag).(bool),
		DiagnosticServer:      getFlagValue(ctx, DiagnosticServerFlag).(int64),
		CompactDb:             getFlagValue(ctx, CompactDbFlag).(bool),
		HasDeletedAccounts:    true,
		KeepDb:                getFlagValue(ctx, KeepDbFlag).(bool),
		KeysNumber:            getFlagValue(ctx, KeysNumberFlag).(int64),
		MaxNumTransactions:    getFlagValue(ctx, MaxNumTransactionsFlag).(int),
		MemoryBreakdown:       getFlagValue(ctx, MemoryBreakdownFlag).(bool),
		MemoryProfile:         getFlagValue(ctx, MemoryProfileFlag).(string),
		NonceRange:            getFlagValue(ctx, NonceRangeFlag).(int),
		TransactionLength:     getFlagValue(ctx, TransactionLengthFlag).(uint64),
		PrimeRandom:           getFlagValue(ctx, RandomizePrimingFlag).(bool),
		RandomSeed:            getFlagValue(ctx, RandomSeedFlag).(int64),
		PrimeThreshold:        getFlagValue(ctx, PrimeThresholdFlag).(int),
		Profile:               getFlagValue(ctx, ProfileFlag).(bool),
		ProfileFile:           getFlagValue(ctx, ProfileFileFlag).(string),
		ProfileInterval:       getFlagValue(ctx, ProfileIntervalFlag).(uint64),
		SkipPriming:           getFlagValue(ctx, SkipPrimingFlag).(bool),
		SkipMetadata:          getFlagValue(ctx, flags.SkipMetadata).(bool),
		ShadowDb:              getFlagValue(ctx, ShadowDb).(bool),
		ShadowImpl:            getFlagValue(ctx, ShadowDbImplementationFlag).(string),
		ShadowVariant:         getFlagValue(ctx, ShadowDbVariantFlag).(string),
		SnapshotDepth:         getFlagValue(ctx, SnapshotDepthFlag).(int),
		StateDbSrc:            getFlagValue(ctx, StateDbSrcFlag).(string),
		AidaDb:                getFlagValue(ctx, AidaDbFlag).(string),
		Output:                getFlagValue(ctx, OutputFlag).(string),
		StateValidationMode:   EqualityCheck,
		UpdateDb:              getFlagValue(ctx, UpdateDbFlag).(string),
		SubstateDb:            getFlagValue(ctx, substate.SubstateDbFlag).(string),
		OperaBinary:           getFlagValue(ctx, OperaBinaryFlag).(string),
		ValuesNumber:          getFlagValue(ctx, ValuesNumberFlag).(int64),
		Validate:              getFlagValue(ctx, ValidateFlag).(bool),
		VmImpl:                getFlagValue(ctx, VmImplementation).(string),
		Workers:               getFlagValue(ctx, substate.WorkersFlag).(int),
		WorldStateDb:          getFlagValue(ctx, WorldStateFlag).(string),
		TraceFile:             getFlagValue(ctx, TraceFileFlag).(string),
		TraceDirectory:        getFlagValue(ctx, TraceDirectoryFlag).(string),
		Trace:                 getFlagValue(ctx, TraceFlag).(bool),
		LogLevel:              getFlagValue(ctx, logger.LogLevelFlag).(string),
		SourceTableName:       getFlagValue(ctx, SourceTableNameFlag).(string),
		TargetDb:              getFlagValue(ctx, TargetDbFlag).(string),
		TrieRootHash:          getFlagValue(ctx, TrieRootHashFlag).(string),
		IncludeStorage:        getFlagValue(ctx, IncludeStorageFlag).(bool),
		ProfileEVMCall:        getFlagValue(ctx, ProfileEVMCallFlag).(bool),
		MicroProfiling:        getFlagValue(ctx, MicroProfilingFlag).(bool),
		BasicBlockProfiling:   getFlagValue(ctx, BasicBlockProfilingFlag).(bool),
		OnlySuccessful:        getFlagValue(ctx, OnlySuccessfulFlag).(bool),
		ProfilingDbName:       getFlagValue(ctx, ProfilingDbNameFlag).(string),
		ChannelBufferSize:     getFlagValue(ctx, ChannelBufferSizeFlag).(int),
		TargetBlock:           getFlagValue(ctx, TargetBlockFlag).(uint64),
		TargetEpoch:           getFlagValue(ctx, TargetEpochFlag).(uint64),
		UpdateBufferSize:      getFlagValue(ctx, UpdateBufferSizeFlag).(uint64),
		StateRootFile:         getFlagValue(ctx, StateRootHashesFlag).(string),
		UpdateOnFailure:       getFlagValue(ctx, UpdateOnFailure).(bool),
		MaxNumErrors:          getFlagValue(ctx, MaxNumErrorsFlag).(int),
		NoHeartbeatLogging:    getFlagValue(ctx, NoHeartbeatLoggingFlag).(bool),
		TrackProgress:         getFlagValue(ctx, TrackProgressFlag).(bool),
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
	}

	return nil
}
