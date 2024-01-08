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

		AidaDb:                 getFlagValue(ctx, AidaDbFlag).(string),
		ArchiveMaxQueryAge:     getFlagValue(ctx, ArchiveMaxQueryAgeFlag).(int),
		ArchiveMode:            getFlagValue(ctx, ArchiveModeFlag).(bool),
		ArchiveQueryRate:       getFlagValue(ctx, ArchiveQueryRateFlag).(int),
		ArchiveVariant:         getFlagValue(ctx, ArchiveVariantFlag).(string),
		BalanceRange:           getFlagValue(ctx, BalanceRangeFlag).(int64),
		BasicBlockProfiling:    getFlagValue(ctx, BasicBlockProfilingFlag).(bool),
		BlockLength:            getFlagValue(ctx, BlockLengthFlag).(uint64),
		CPUProfile:             getFlagValue(ctx, CpuProfileFlag).(string),
		CPUProfilePerInterval:  getFlagValue(ctx, CpuProfilePerIntervalFlag).(bool),
		Cache:                  getFlagValue(ctx, CacheFlag).(int),
		CarmenSchema:           getFlagValue(ctx, CarmenSchemaFlag).(int),
		ChainID:                ChainID(getFlagValue(ctx, ChainIDFlag).(int)),
		ChannelBufferSize:      getFlagValue(ctx, ChannelBufferSizeFlag).(int),
		CompactDb:              getFlagValue(ctx, CompactDbFlag).(bool),
		ContinueOnFailure:      getFlagValue(ctx, ContinueOnFailureFlag).(bool),
		ContractNumber:         getFlagValue(ctx, ContractNumberFlag).(int64),
		DbComponent:            getFlagValue(ctx, DbComponentFlag).(string),
		DbImpl:                 getFlagValue(ctx, StateDbImplementationFlag).(string),
		DbLogging:              getFlagValue(ctx, StateDbLoggingFlag).(string),
		DbTmp:                  getFlagValue(ctx, DbTmpFlag).(string),
		DbVariant:              getFlagValue(ctx, StateDbVariantFlag).(string),
		Debug:                  getFlagValue(ctx, TraceDebugFlag).(bool),
		DebugFrom:              getFlagValue(ctx, DebugFromFlag).(uint64),
		DeleteSourceDbs:        getFlagValue(ctx, DeleteSourceDbsFlag).(bool),
		DeletionDb:             getFlagValue(ctx, DeletionDbFlag).(string),
		DiagnosticServer:       getFlagValue(ctx, DiagnosticServerFlag).(int64),
		ErrorLogging:           getFlagValue(ctx, ErrorLoggingFlag).(string),
		Genesis:                getFlagValue(ctx, GenesisFlag).(string),
		IncludeStorage:         getFlagValue(ctx, IncludeStorageFlag).(bool),
		KeepDb:                 getFlagValue(ctx, KeepDbFlag).(bool),
		KeysNumber:             getFlagValue(ctx, KeysNumberFlag).(int64),
		LogLevel:               getFlagValue(ctx, logger.LogLevelFlag).(string),
		MaxNumErrors:           getFlagValue(ctx, MaxNumErrorsFlag).(int),
		MaxNumTransactions:     getFlagValue(ctx, MaxNumTransactionsFlag).(int),
		MemoryBreakdown:        getFlagValue(ctx, MemoryBreakdownFlag).(bool),
		MemoryProfile:          getFlagValue(ctx, MemoryProfileFlag).(string),
		MicroProfiling:         getFlagValue(ctx, MicroProfilingFlag).(bool),
		NoHeartbeatLogging:     getFlagValue(ctx, NoHeartbeatLoggingFlag).(bool),
		NonceRange:             getFlagValue(ctx, NonceRangeFlag).(int),
		OnlySuccessful:         getFlagValue(ctx, OnlySuccessfulFlag).(bool),
		OperaBinary:            getFlagValue(ctx, OperaBinaryFlag).(string),
		OperaDb:                getFlagValue(ctx, OperaDbFlag).(string),
		Output:                 getFlagValue(ctx, OutputFlag).(string),
		OverwriteRunId:         getFlagValue(ctx, OverwriteRunIdFlag).(string),
		PrimeRandom:            getFlagValue(ctx, RandomizePrimingFlag).(bool),
		PrimeThreshold:         getFlagValue(ctx, PrimeThresholdFlag).(int),
		Profile:                getFlagValue(ctx, ProfileFlag).(bool),
		ProfileBlocks:          getFlagValue(ctx, ProfileBlocksFlag).(bool),
		ProfileDB:              getFlagValue(ctx, ProfileDBFlag).(string),
		ProfileDepth:           getFlagValue(ctx, ProfileDepthFlag).(int),
		ProfileEVMCall:         getFlagValue(ctx, ProfileEVMCallFlag).(bool),
		ProfileFile:            getFlagValue(ctx, ProfileFileFlag).(string),
		ProfileInterval:        getFlagValue(ctx, ProfileIntervalFlag).(uint64),
		ProfileSqlite3:         getFlagValue(ctx, ProfileSqlite3Flag).(string),
		ProfilingDbName:        getFlagValue(ctx, ProfilingDbNameFlag).(string),
		RandomSeed:             getFlagValue(ctx, RandomSeedFlag).(int64),
		RegisterRun:            getFlagValue(ctx, RegisterRunFlag).(string),
		RpcRecordingFile:       getFlagValue(ctx, RpcRecordingFileFlag).(string),
		ShadowDb:               getFlagValue(ctx, ShadowDb).(bool),
		ShadowImpl:             getFlagValue(ctx, ShadowDbImplementationFlag).(string),
		ShadowVariant:          getFlagValue(ctx, ShadowDbVariantFlag).(string),
		SkipMetadata:           getFlagValue(ctx, flags.SkipMetadata).(bool),
		SkipPriming:            getFlagValue(ctx, SkipPrimingFlag).(bool),
		SkipStateHashScrapping: getFlagValue(ctx, SkipStateHashScrappingFlag).(bool),
		SnapshotDepth:          getFlagValue(ctx, SnapshotDepthFlag).(int),
		SourceTableName:        getFlagValue(ctx, SourceTableNameFlag).(string),
		SrcDbReadonly:          false,
		StateDbSrc:             getFlagValue(ctx, StateDbSrcFlag).(string),
		StateValidationMode:    EqualityCheck,
		SubstateDb:             getFlagValue(ctx, substate.SubstateDbFlag).(string),
		SyncPeriodLength:       getFlagValue(ctx, SyncPeriodLengthFlag).(uint64),
		TargetBlock:            getFlagValue(ctx, TargetBlockFlag).(uint64),
		TargetDb:               getFlagValue(ctx, TargetDbFlag).(string),
		TargetEpoch:            getFlagValue(ctx, TargetEpochFlag).(uint64),
		Trace:                  getFlagValue(ctx, TraceFlag).(bool),
		TraceDirectory:         getFlagValue(ctx, TraceDirectoryFlag).(string),
		TraceFile:              getFlagValue(ctx, TraceFileFlag).(string),
		TrackProgress:          getFlagValue(ctx, TrackProgressFlag).(bool),
		TransactionLength:      getFlagValue(ctx, TransactionLengthFlag).(uint64),
		TrieRootHash:           getFlagValue(ctx, TrieRootHashFlag).(string),
		UpdateBufferSize:       getFlagValue(ctx, UpdateBufferSizeFlag).(uint64),
		UpdateDb:               getFlagValue(ctx, UpdateDbFlag).(string),
		UpdateOnFailure:        getFlagValue(ctx, UpdateOnFailureFlag).(bool),
		UpdateType:             getFlagValue(ctx, UpdateTypeFlag).(string),
		Validate:               getFlagValue(ctx, ValidateFlag).(bool),
		ValidateStateHashes:    getFlagValue(ctx, ValidateStateHashesFlag).(bool),
		ValidateTxState:        getFlagValue(ctx, ValidateTxStateFlag).(bool),
		ValuesNumber:           getFlagValue(ctx, ValuesNumberFlag).(int64),
		VmImpl:                 getFlagValue(ctx, VmImplementation).(string),
		Workers:                getFlagValue(ctx, substate.WorkersFlag).(int),
		WorldStateDb:           getFlagValue(ctx, WorldStateFlag).(string),
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
