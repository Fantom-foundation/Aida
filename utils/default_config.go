package utils

import (
	"fmt"
	"reflect"

	"github.com/Fantom-foundation/Aida/cmd/util-db/flags"
	"github.com/Fantom-foundation/Aida/logger"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// createConfigFromFlags returns Config instance with user specified values or the default ones
func createConfigFromFlags(ctx *cli.Context) (*Config, map[string]bool, error) {
	cfg := &Config{
		AppName:     ctx.App.HelpName,
		CommandName: ctx.Command.Name,
	}

	// string of this map has to exactly match the name of the field in Config struct
	cfgFlags := map[string]interface{}{
		"AidaDb":                 AidaDbFlag,
		"ArchiveMaxQueryAge":     ArchiveMaxQueryAgeFlag,
		"ArchiveMode":            ArchiveModeFlag,
		"ArchiveQueryRate":       ArchiveQueryRateFlag,
		"ArchiveVariant":         ArchiveVariantFlag,
		"BalanceRange":           BalanceRangeFlag,
		"BasicBlockProfiling":    BasicBlockProfilingFlag,
		"BlockLength":            BlockLengthFlag,
		"Cache":                  CacheFlag,
		"CarmenSchema":           CarmenSchemaFlag,
		"ChainID":                ChainIDFlag,
		"ChannelBufferSize":      ChannelBufferSizeFlag,
		"CompactDb":              CompactDbFlag,
		"ContinueOnFailure":      ContinueOnFailureFlag,
		"ContractNumber":         ContractNumberFlag,
		"CPUProfile":             CpuProfileFlag,
		"CPUProfilePerInterval":  CpuProfilePerIntervalFlag,
		"DbComponent":            DbComponentFlag,
		"DbImpl":                 StateDbImplementationFlag,
		"DbLogging":              StateDbLoggingFlag,
		"DbTmp":                  DbTmpFlag,
		"DbVariant":              StateDbVariantFlag,
		"Debug":                  TraceDebugFlag,
		"DebugFrom":              DebugFromFlag,
		"DeleteSourceDbs":        DeleteSourceDbsFlag,
		"DeletionDb":             DeletionDbFlag,
		"DiagnosticServer":       DiagnosticServerFlag,
		"ErrorLogging":           ErrorLoggingFlag,
		"Genesis":                GenesisFlag,
		"IncludeStorage":         IncludeStorageFlag,
		"KeepDb":                 KeepDbFlag,
		"KeysNumber":             KeysNumberFlag,
		"LogLevel":               logger.LogLevelFlag,
		"MaxNumErrors":           MaxNumErrorsFlag,
		"MaxNumTransactions":     MaxNumTransactionsFlag,
		"MemoryBreakdown":        MemoryBreakdownFlag,
		"MemoryProfile":          MemoryProfileFlag,
		"MicroProfiling":         MicroProfilingFlag,
		"NoHeartbeatLogging":     NoHeartbeatLoggingFlag,
		"NonceRange":             NonceRangeFlag,
		"OnlySuccessful":         OnlySuccessfulFlag,
		"OperaBinary":            OperaBinaryFlag,
		"OperaDb":                OperaDbFlag,
		"Output":                 OutputFlag,
		"PrimeRandom":            RandomizePrimingFlag,
		"PrimeThreshold":         PrimeThresholdFlag,
		"Profile":                ProfileFlag,
		"ProfileBlocks":          ProfileBlocksFlag,
		"ProfileDB":              ProfileDBFlag,
		"ProfileDepth":           ProfileDepthFlag,
		"ProfileEVMCall":         ProfileEVMCallFlag,
		"ProfileFile":            ProfileFileFlag,
		"ProfileInterval":        ProfileIntervalFlag,
		"ProfileSqlite3":         ProfileSqlite3Flag,
		"ProfilingDbName":        ProfilingDbNameFlag,
		"RandomSeed":             RandomSeedFlag,
		"RpcRecordingFile":       RpcRecordingFileFlag,
		"ShadowDb":               ShadowDb,
		"ShadowImpl":             ShadowDbImplementationFlag,
		"ShadowVariant":          ShadowDbVariantFlag,
		"SkipMetadata":           flags.SkipMetadata,
		"SkipPriming":            SkipPrimingFlag,
		"SkipStateHashScrapping": SkipStateHashScrappingFlag,
		"SnapshotDepth":          SnapshotDepthFlag,
		"SourceTableName":        SourceTableNameFlag,
		"StateDbSrc":             StateDbSrcFlag,
		"SubstateDb":             substate.SubstateDbFlag,
		"SyncPeriodLength":       SyncPeriodLengthFlag,
		"TargetBlock":            TargetBlockFlag,
		"TargetDb":               TargetDbFlag,
		"TargetEpoch":            TargetEpochFlag,
		"Trace":                  TraceFlag,
		"TraceDirectory":         TraceDirectoryFlag,
		"TraceFile":              TraceFileFlag,
		"TrackProgress":          TrackProgressFlag,
		"TransactionLength":      TransactionLengthFlag,
		"TrieRootHash":           TrieRootHashFlag,
		"UpdateBufferSize":       UpdateBufferSizeFlag,
		"UpdateDb":               UpdateDbFlag,
		"UpdateOnFailure":        UpdateOnFailure,
		"UpdateType":             UpdateTypeFlag,
		"Validate":               ValidateFlag,
		"ValidateStateHashes":    ValidateStateHashesFlag,
		"ValidateTxState":        ValidateTxStateFlag,
		"ValuesNumber":           ValuesNumberFlag,
		"VmImpl":                 VmImplementation,
		"Workers":                substate.WorkersFlag,
		"WorldStateDb":           WorldStateFlag,
	}

	cfgValue := reflect.ValueOf(cfg).Elem()

	specifiedFlags := make(map[string]bool)

	for cfgName, flag := range cfgFlags {
		value, isSpecified, flagName := getFlagValue(ctx, flag)
		if isSpecified {
			specifiedFlags[flagName] = true
		}

		field := cfgValue.FieldByName(cfgName)
		if !field.IsValid() {
			return nil, nil, fmt.Errorf("field %s is not valid", flagName)
		}
		if !field.CanSet() {
			return nil, nil, fmt.Errorf("field %s cannot be set", flagName)
		}

		field.Set(reflect.ValueOf(value))
	}

	return cfg, specifiedFlags, nil
}

// getFlagValue returns value specified by user if flag is present in cli context, otherwise return default flag value
func getFlagValue(ctx *cli.Context, flag interface{}) (interface{}, bool, string) {
	cmdFlags := ctx.Command.Flags
	for _, cmdFlag := range cmdFlags {
		switch f := flag.(type) {
		case cli.IntFlag:
			if cmdFlag.Names()[0] == f.Name {
				if cmdFlag.Names()[0] == ChainIDFlag.Name {
					return ChainID(ctx.Int(f.Name)), true, f.Name
				}
				return ctx.Int(f.Name), true, f.Name
			}

		case cli.Uint64Flag:
			if cmdFlag.Names()[0] == UpdateBufferSizeFlag.Name {
				return ctx.Uint64(f.Name) * 1_000_000, true, f.Name
			} else if cmdFlag.Names()[0] == f.Name {
				return ctx.Uint64(f.Name), true, f.Name
			}

		case cli.Int64Flag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.Int64(f.Name), true, f.Name
			}

		case cli.StringFlag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.String(f.Name), true, f.Name
			}

		case cli.PathFlag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.Path(f.Name), true, f.Name
			}

		case cli.BoolFlag:
			if cmdFlag.Names()[0] == f.Name {
				return ctx.Bool(f.Name), true, f.Name
			}
		}
	}

	// If flag not found, return the default value of the flag and false
	switch f := flag.(type) {
	case cli.IntFlag:
		return f.Value, false, f.Name
	case cli.Uint64Flag:
		return f.Value, false, f.Name
	case cli.Int64Flag:
		return f.Value, false, f.Name
	case cli.StringFlag:
		return f.Value, false, f.Name
	case cli.PathFlag:
		return f.Value, false, f.Name
	case cli.BoolFlag:
		return f.Value, false, f.Name
	}
	return nil, false, ""
}
