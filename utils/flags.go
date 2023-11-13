package utils

import (
	"github.com/urfave/cli/v2"
)

// Command line options for common flags in record and replay.
var (
	RpcRecordingFileFlag = cli.PathFlag{
		Name:    "rpc-recording",
		Usage:   "Path to source file with recorded API data",
		Aliases: []string{"r"},
	}
	ArchiveModeFlag = cli.BoolFlag{
		Name:  "archive",
		Usage: "set node type to archival mode. If set, the node keep all the EVM state history; otherwise the state history will be pruned.",
	}
	ArchiveQueryRateFlag = cli.IntFlag{
		Name:  "archive-query-rate",
		Usage: "sets the number of queries send to the archive per second, disabled if 0 or negative",
	}
	ArchiveMaxQueryAgeFlag = cli.IntFlag{
		Name:  "archive-max-query-age",
		Usage: "sets an upper limit for the number of blocks an archive query may be lagging behind the head block",
		Value: 100_000,
	}
	ArchiveVariantFlag = cli.StringFlag{
		Name:  "archive-variant",
		Usage: "set the archive implementation variant for the selected DB implementation, ignored if not running in archive mode",
	}
	BlockLengthFlag = cli.Uint64Flag{
		Name:  "block-length",
		Usage: "defines the number of transactions per block",
		Value: 10,
	}
	BalanceRangeFlag = cli.Int64Flag{
		Name:  "balance-range",
		Usage: "sets the balance range of the stochastic simulation",
		Value: 1000000,
	}
	CarmenSchemaFlag = cli.IntFlag{
		Name:  "carmen-schema",
		Usage: "select the DB schema used by Carmen's current state DB",
		Value: 3,
	}
	ChainIDFlag = cli.IntFlag{
		Name:  "chainid",
		Usage: "ChainID for replayer",
	}
	CacheFlag = cli.IntFlag{
		Name:  "cache",
		Usage: "Cache limit for StateDb or Priming",
		Value: 8192,
	}
	ContinueOnFailureFlag = cli.BoolFlag{
		Name:  "continue-on-failure",
		Usage: "continue execute after validation failure detected",
	}
	CpuProfileFlag = cli.StringFlag{
		Name:  "cpu-profile",
		Usage: "enables CPU profiling",
	}
	CpuProfilePerIntervalFlag = cli.BoolFlag{
		Name:  "cpu-profile-per-interval",
		Usage: "enables CPU profiling for individual 100k intervals",
	}
	DebugFromFlag = cli.Uint64Flag{
		Name:  "debug-from",
		Usage: "sets the first block to print trace debug",
		Value: 0,
	}
	DeletionDbFlag = cli.PathFlag{
		Name:  "deletion-db",
		Usage: "sets the directory containing deleted accounts database",
	}
	DiagnosticServerFlag = cli.Int64Flag{
		Name:  "diagnostic-port",
		Usage: "enable hosting of a realtime diagnostic server by providing a port",
		Value: 0,
	}
	KeepDbFlag = cli.BoolFlag{
		Name:  "keep-db",
		Usage: "if set, statedb is not deleted after run",
	}
	MemoryProfileFlag = cli.StringFlag{
		Name:  "memory-profile",
		Usage: "enables memory allocation profiling",
	}
	SyncPeriodLengthFlag = cli.Uint64Flag{
		Name:  "sync-period",
		Usage: "defines the number of blocks per sync-period",
		Value: 300,
	}
	MemoryBreakdownFlag = cli.BoolFlag{
		Name:  "memory-breakdown",
		Usage: "enables printing of memory usage breakdown",
	}
	NonceRangeFlag = cli.IntFlag{
		Name:  "nonce-range",
		Usage: "sets nonce range for stochastic simulation",
		Value: 1000000,
	}
	ProfileFlag = cli.BoolFlag{
		Name:  "profile",
		Usage: "enable profiling",
	}
	ProfileFileFlag = cli.StringFlag{
		Name:  "profile-file",
		Usage: "output file containing profiling data",
	}
	ProfileIntervalFlag = cli.Uint64Flag{
		Name:  "profile-interval",
		Usage: "Frequency of logging block statistics",
		Value: 1_000_000_000,
	}
	QuietFlag = cli.BoolFlag{
		Name:  "quiet",
		Usage: "disable progress report",
	}
	RandomizePrimingFlag = cli.BoolFlag{
		Name:  "prime-random",
		Usage: "randomize order of accounts in StateDB priming",
	}
	PrimeThresholdFlag = cli.IntFlag{
		Name:  "prime-threshold",
		Usage: "set number of accounts written to stateDB before applying pending state updates",
		Value: 0,
	}
	RandomSeedFlag = cli.Int64Flag{
		Name:  "random-seed",
		Usage: "Set random seed",
		Value: -1,
	}
	SkipPrimingFlag = cli.BoolFlag{
		Name:  "skip-priming",
		Usage: "if set, DB priming should be skipped; most useful with the 'memory' DB implementation",
	}
	StateDbImplementationFlag = cli.StringFlag{
		Name:  "db-impl",
		Usage: "select state DB implementation",
		Value: "geth",
	}
	StateDbVariantFlag = cli.StringFlag{
		Name:  "db-variant",
		Usage: "select a state DB variant",
		Value: "",
	}
	StateDbSrcFlag = cli.PathFlag{
		Name:  "db-src",
		Usage: "sets the directory contains source state DB data",
	}
	DbTmpFlag = cli.PathFlag{
		Name:  "db-tmp",
		Usage: "sets the temporary directory where to place DB data; uses system default if empty",
	}
	StateDbLoggingFlag = cli.BoolFlag{
		Name:  "db-logging",
		Usage: "enable logging of all DB operations",
	}
	ShadowDb = cli.BoolFlag{
		Name:  "shadow-db",
		Usage: "use this flag when using an existing ShadowDb",
		Value: false,
	}
	ShadowDbImplementationFlag = cli.StringFlag{
		Name:  "db-shadow-impl",
		Usage: "select state DB implementation to shadow the prime DB implementation",
		Value: "",
	}
	ShadowDbVariantFlag = cli.StringFlag{
		Name:  "db-shadow-variant",
		Usage: "select a state DB variant to shadow the prime DB implementation",
		Value: "",
	}
	TraceFlag = cli.BoolFlag{
		Name:  "trace",
		Usage: "enable tracing",
	}
	TraceDebugFlag = cli.BoolFlag{
		Name:  "trace-debug",
		Usage: "enable debug output for tracing",
	}
	TraceFileFlag = cli.PathFlag{
		Name:  "trace-file",
		Usage: "set storage trace's output directory",
		Value: "./",
	}
	TraceDirectoryFlag = cli.PathFlag{
		Name:  "trace-dir",
		Usage: "set storage trace directory",
	}
	UpdateDbFlag = cli.PathFlag{
		Name:  "update-db",
		Usage: "set update-set database directory",
	}
	UpdateTypeFlag = cli.StringFlag{
		Name:  "update-type",
		Usage: "select update type (\"stable\" or \"nightly\")",
		Value: "stable",
	}
	OperaBinaryFlag = cli.PathFlag{
		Name:  "opera-binary",
		Usage: "opera binary path",
	}
	ValidateFlag = cli.BoolFlag{
		Name:  "validate",
		Usage: "enables validation",
	}
	ValidateTxStateFlag = cli.BoolFlag{
		Name:  "validate-tx",
		Usage: "enables transaction state validation",
	}
	ValidateWorldStateFlag = cli.BoolFlag{
		Name:  "validate-ws",
		Usage: "enables end-state validation",
	}
	VmImplementation = cli.StringFlag{
		Name:  "vm-impl",
		Usage: "select VM implementation",
		Value: "geth",
	}
	WorldStateFlag = cli.PathFlag{
		Name:  "world-state",
		Usage: "world state snapshot database path",
	}
	MaxNumTransactionsFlag = cli.IntFlag{
		Name:  "max-tx",
		Usage: "limit the maximum number of processed transactions, default: unlimited",
		Value: -1,
	}
	OutputFlag = cli.PathFlag{
		Name:  "output",
		Usage: "output path",
	}
	PortFlag = cli.StringFlag{
		Name:        "port",
		Aliases:     []string{"v"},
		Usage:       "enable visualization on `PORT`",
		DefaultText: "8080",
	}
	DeleteSourceDbsFlag = cli.BoolFlag{
		Name:  "delete-source-dbs",
		Usage: "delete source databases while merging into one database",
		Value: false,
	}
	CompactDbFlag = cli.BoolFlag{
		Name:  "compact",
		Usage: "compact target database",
		Value: false,
	}
	AidaDbFlag = cli.PathFlag{
		Name:     "aida-db",
		Usage:    "set substate, updateset and deleted accounts directory",
		Required: true,
	}
	ContractNumberFlag = cli.Int64Flag{
		Name:  "num-contracts",
		Usage: "Number of contracts to create",
		Value: 1_000,
	}
	KeysNumberFlag = cli.Int64Flag{
		Name:  "num-keys",
		Usage: "Number of keys to generate",
		Value: 1_000,
	}
	ValuesNumberFlag = cli.Int64Flag{
		Name:  "num-values",
		Usage: "Number of values to generate",
		Value: 1_000,
	}
	TransactionLengthFlag = cli.Uint64Flag{
		Name:  "transaction-length",
		Usage: "Determines indirectly the length of a transaction",
		Value: 10,
	}
	SnapshotDepthFlag = cli.IntFlag{
		Name:  "snapshot-depth",
		Usage: "Depth of snapshot history",
		Value: 100,
	}
	OperaDbFlag = cli.PathFlag{
		Name:    "db",
		Aliases: []string{"datadir"},
		Usage:   "Path to the opera database",
	}
	GenesisFlag = cli.PathFlag{
		Name:  "genesis",
		Usage: "Path to genesis file",
	}
	SourceTableNameFlag = cli.StringFlag{
		Name:  "source-table",
		Usage: "name of the database table to be used",
		Value: "main",
	}
	TargetDbFlag = cli.PathFlag{
		Name:  "target-db",
		Usage: "target database path",
	}
	TrieRootHashFlag = cli.StringFlag{
		Name:  "root",
		Usage: "state trie root hash to be analysed",
	}
	IncludeStorageFlag = cli.BoolFlag{
		Name:  "include-storage",
		Usage: "display full storage content",
	}
	ProfileEVMCallFlag = cli.BoolFlag{
		Name:  "profiling-call",
		Usage: "enable profiling for EVM call",
	}
	MicroProfilingFlag = cli.BoolFlag{
		Name:  "micro-profiling",
		Usage: "enable micro-profiling of EVM",
	}
	BasicBlockProfilingFlag = cli.BoolFlag{
		Name:  "basic-block-profiling",
		Usage: "enable profiling of basic block",
	}
	OnlySuccessfulFlag = cli.BoolFlag{
		Name:  "only-successful",
		Usage: "only runs transactions that have been successful",
	}
	ProfilingDbNameFlag = cli.StringFlag{
		Name:  "profiling-db-name",
		Usage: "set a database name for storing micro-profiling results",
		Value: "./profiling.db",
	}
	ChannelBufferSizeFlag = cli.IntFlag{
		Name:  "buffer-size",
		Usage: "set a buffer size for profiling channel",
		Value: 100_000,
	}
	UpdateBufferSizeFlag = cli.Uint64Flag{
		Name:  "update-buffer-size",
		Usage: "buffer size for holding update set in MB",
		Value: 1_000_000,
	}
	TargetBlockFlag = cli.Uint64Flag{
		Name:    "target-block",
		Aliases: []string{"block", "blk"},
		Usage:   "target block ID",
		Value:   0,
	}
	TargetEpochFlag = cli.Uint64Flag{
		Name:    "target-epoch",
		Aliases: []string{"epoch"},
		Usage:   "target epoch ID",
		Value:   0,
	}
	MaxNumErrorsFlag = cli.IntFlag{
		Name:  "max-errors",
		Usage: "maximum number of errors when ContinueOnFailure is enabled, default is 50",
		Value: 50,
	}
	UpdateOnFailure = cli.BoolFlag{
		Name:  "update-on-failure",
		Usage: "if enabled and continue-on-failure is also enabled, this corrects any error found in StateDb",
		Value: true,
	}
	SkipStateHashScrappingFlag = cli.BoolFlag{
		Name:  "skip-state-hash-scrapping",
		Usage: "if enabled, then state-hashes are not loaded from rpc",
		Value: false,
	}
	NoHeartbeatLoggingFlag = cli.BoolFlag{
		Name:  "no-heartbeat-logging",
		Usage: "disables heartbeat logging",
	}
	TrackProgressFlag = cli.BoolFlag{
		Name:  "track-progress",
		Usage: "enables track progress logging",
	}
	ValidateStateHashesFlag = cli.BoolFlag{
		Name:  "validate-state-hash",
		Usage: "enables state hash validation",
	}
	ProfileBlocksFlag = cli.BoolFlag{
		Name:  "profile-blocks",
		Usage: "enables block profiling",
	}
	ProfileDBFlag = cli.PathFlag{
		Name:  "profile-db",
		Usage: "defines path to profile-db",
		Value: "/var/opera/Aida/profile.db",
	}
)
