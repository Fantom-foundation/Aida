package utils

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/logger"
	_ "github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/c2h5oh/datasize"
	"github.com/ethereum/go-ethereum/core/rawdb"
	_ "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

type ArgumentMode int
type ChainID int
type ChainIDs []ChainID

// An enums of argument modes used by trace subcommands
const (
	BlockRangeArgs          ArgumentMode = iota // requires 2 arguments: first block and last block
	BlockRangeArgsProfileDB                     // requires 3 arguments: first block, last block and profile db
	LastBlockArg                                // requires 1 argument: last block
	NoArgs                                      // requires no arguments
)

const (
	UnknownChainID ChainID = 0
	MainnetChainID ChainID = 250
	TestnetChainID ChainID = 4002
)

var AvailableChainIDs = ChainIDs{MainnetChainID, TestnetChainID}

const (
	aidaDbRepositoryMainnetUrl = "https://aida.repository.fantom.network"
	aidaDbRepositoryTestnetUrl = "https://aida.testnet.repository.fantom.network"
)

var (
	FirstOperaBlock     uint64 // id of the first block in substate
	AidaDbRepositoryUrl string // url of the Aida DB repository
)

// Type of validation performs on stateDB during Tx processing.
type ValidationMode int

const (
	SubsetCheck   ValidationMode = iota // confirms whether a substate is contained in stateDB.
	EqualityCheck                       // confirms whether a substate and StateDB are identical.
)

var hardForksMainnet = map[string]uint64{
	"zero":   0,
	"opera":  4_564_026,
	"berlin": 37_455_223,
	"london": 37_534_833,
}

var hardForksTestnet = map[string]uint64{
	"zero":   0,
	"opera":  479_327,
	"berlin": 1_559_470,
	"london": 7_513_335,
}

// special transaction number for pseudo transactions
const PseudoTx = 99999

// GitCommit represents the GitHub commit hash the app was built from.
var GitCommit = "0000000000000000000000000000000000000000"

// Command line options for common flags in record and replay.
var (
	APIRecordingSrcFileFlag = cli.PathFlag{
		Name:    "api-recording",
		Usage:   "Path to source file with recorded API data",
		Aliases: []string{"r"},
	}
	ArchiveModeFlag = cli.BoolFlag{
		Name:  "archive",
		Usage: "set node type to archival mode. If set, the node keep all the EVM state history; otherwise the state history will be pruned.",
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
		Value: 0,
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
	DebugFromFlag = cli.Uint64Flag{
		Name:  "debug-from",
		Usage: "sets the first block to print trace debug",
		Value: 0,
	}
	DeletionDbFlag = cli.PathFlag{
		Name:  "deletion-db",
		Usage: "sets the directory containing deleted accounts database",
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
	OperaDatadirFlag = cli.PathFlag{
		Name:  "datadir",
		Usage: "opera datadir directory",
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
		Name:  "aida-db",
		Usage: "set substate, updateset and deleted accounts directory",
	}
	ErigonBatchSizeFlag = cli.StringFlag{
		Name:  "erigonbatchsize",
		Usage: "Batch size for the execution stage",
		Value: "512M",
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
	DbFlag = cli.PathFlag{
		Name:  "db",
		Usage: "Path to the database",
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
)

// Config represents execution configuration for replay command.
type Config struct {
	AppName     string
	CommandName string

	First uint64 // first block
	Last  uint64 // last block

	APIRecordingSrcFile string            // path to source file with recorded API data
	ArchiveMode         bool              // enable archive mode
	ArchiveVariant      string            // selects the implementation variant of the archive
	BlockLength         uint64            // length of a block in number of transactions
	BalanceRange        int64             // balance range for stochastic simulation/replay
	CarmenSchema        int               // the current DB schema ID to use in Carmen
	ChainID             ChainID           // Blockchain ID (mainnet: 250/testnet: 4002)
	Cache               int               // Cache for StateDb or Priming
	ContinueOnFailure   bool              // continue validation when an error detected
	ContractNumber      int64             // number of contracts to create
	CompactDb           bool              // compact database after merging
	CopySrcDb           bool              // if true, make a copy the source statedb
	CPUProfile          string            // pprof cpu profile output file name
	Db                  string            // path to database
	DbTmp               string            // path to temporary database
	DbImpl              string            // storage implementation
	Genesis             string            // genesis file
	DbVariant           string            // database variant
	DbLogging           bool              // set to true if all DB operations should be logged
	Debug               bool              // enable trace debug flag
	DeleteSourceDbs     bool              // delete source databases
	DebugFrom           uint64            // the first block to print trace debug
	DeletionDb          string            // directory of deleted account database
	Quiet               bool              // disable progress report flag
	SyncPeriodLength    uint64            // length of a sync-period in number of blocks
	HasDeletedAccounts  bool              // true if DeletionDb is not empty; otherwise false
	KeepDb              bool              // set to true if db is kept after run
	KeysNumber          int64             // number of keys to generate
	MaxNumTransactions  int               // the maximum number of processed transactions
	MemoryBreakdown     bool              // enable printing of memory breakdown
	MemoryProfile       string            // capture the memory heap profile into the file
	NonceRange          int               // nonce range for stochastic simulation/replay
	TransactionLength   uint64            // determines indirectly the length of a transaction
	PrimeRandom         bool              // enable randomized priming
	PrimeThreshold      int               // set account threshold before commit
	Profile             bool              // enable micro profiling
	ProfileFile         string            // output file containing profiling result
	ProfileInterval     uint64            // interval of printing profile result
	RandomSeed          int64             // set random seed for stochastic testing
	SkipPriming         bool              // skip priming of the state DB
	SkipMetadata        bool              // skip metadata insert/getting into AidaDb
	ShadowDb            bool              // defines we want to open an existing db as shadow
	ShadowImpl          string            // implementation of the shadow DB to use, empty if disabled
	ShadowVariant       string            // database variant of the shadow DB to be used
	StateDbSrc          string            // directory to load an existing State DB data
	AidaDb              string            // directory to profiling database containing substate, update, delete accounts data
	StateValidationMode ValidationMode    // state validation mode
	UpdateDb            string            // update-set directory
	Output              string            // output directory for aida-db patches or path to events.json file in stochastic generation
	SnapshotDepth       int               // depth of snapshot history
	SubstateDb          string            // substate directory
	OperaDatadir        string            // source opera directory
	Validate            bool              // validate validate aida-db
	ValidateTxState     bool              // validate stateDB before and after transaction
	ValidateWorldState  bool              // validate stateDB before and after replay block range
	ValuesNumber        int64             // number of values to generate
	VmImpl              string            // vm implementation (geth/lfvm)
	WorldStateDb        string            // path to worldstate
	Workers             int               // number of worker threads
	TraceFile           string            // name of trace file
	TraceDirectory      string            // name of trace directory
	Trace               bool              // trace flag
	LogLevel            string            // level of the logging of the app action
	SourceTableName     string            // represents the name of a source DB table
	TargetDb            string            // represents the path of a target DB
	TrieRootHash        string            // represents a hash of a state trie root to be decoded
	IncludeStorage      bool              // represents a flag for contract storage inclusion in an operation
	ProfileEVMCall      bool              // enable profiling for EVM call
	MicroProfiling      bool              // enable micro-profiling of EVM
	BasicBlockProfiling bool              // enable profiling of basic block
	OnlySuccessful      bool              // only runs transactions that have been successful
	ProfilingDbName     string            // set a database name for storing micro-profiling results
	ChannelBufferSize   int               // set a buffer size for profiling channel
	TargetBlock         uint64            // represents the ID of target block to be reached by state evolve process or in dump state
	UpdateBufferSize    uint64            // cache size in Bytes
	ErigonBatchSize     datasize.ByteSize // erigon batch size for runVM
	ProfileDB           string            // profile db for parallel transaction execution

}

// GetChainConfig returns chain configuration of either mainnet or testnets.
func GetChainConfig(chainID ChainID) *params.ChainConfig {
	chainConfig := params.AllEthashProtocolChanges
	chainConfig.ChainID = big.NewInt(int64(chainID))
	if chainID == MainnetChainID {
		// mainnet chainID 250
		chainConfig.BerlinBlock = new(big.Int).SetUint64(hardForksMainnet["berlin"])
		chainConfig.LondonBlock = new(big.Int).SetUint64(hardForksMainnet["london"])
	} else if chainID == TestnetChainID {
		// testnet chainID 4002
		chainConfig.BerlinBlock = new(big.Int).SetUint64(hardForksTestnet["berlin"])
		chainConfig.LondonBlock = new(big.Int).SetUint64(hardForksTestnet["london"])
	} else {
		log.Fatalf("unknown chain id %v", chainID)
	}
	return chainConfig
}

func setFirstBlockFromChainID(chainID ChainID) {
	if chainID == MainnetChainID {
		FirstOperaBlock = hardForksMainnet["opera"]
	} else if chainID == TestnetChainID {
		FirstOperaBlock = hardForksTestnet["opera"]
	} else {
		log.Fatalf("unknown chain id %v", chainID)
	}
}

// NewConfig creates and initializes Config with commandline arguments.
func NewConfig(ctx *cli.Context, mode ArgumentMode) (*Config, error) {
	log := logger.NewLogger(ctx.String(logger.LogLevelFlag.Name), "Config")

	var (
		first, last uint64
		profileDB   string
		chainId     ChainID
	)

	chainId = ChainID(ctx.Int(ChainIDFlag.Name))

	// first look for chainId since we need it for verbal block indication
	if chainId == UnknownChainID {
		log.Warningf("ChainID (--%v) was not set; looking for it in AidaDb", ChainIDFlag.Name)

		// we check if AidaDb was set with err == nil
		if aidaDb, err := rawdb.NewLevelDBDatabase(ctx.String(AidaDbFlag.Name), 1024, 100, "profiling", true); err == nil {
			md := NewAidaDbMetadata(aidaDb, ctx.String(logger.LogLevelFlag.Name))

			chainId = ChainID(md.GetChainID())

			if err = aidaDb.Close(); err != nil {
				return nil, fmt.Errorf("cannot close db; %v", err)
			}
		}

		if chainId == UnknownChainID {
			log.Warningf("ChainID was neither specified with flag (--%v) nor was found in AidaDb (%v); setting default value for mainnet", ChainIDFlag.Name, ctx.String(AidaDbFlag.Name))
			chainId = MainnetChainID
		} else {
			log.Noticef("Found chainId (%v) in AidaDb", chainId)
		}

	}

	var argErr error
	switch mode {
	case BlockRangeArgs:
		// try to extract block range from db metadata
		mdFirst, mdLast, err := getMdBlockRange(ctx)
		if err != nil {
			return nil, err
		}

		// process arguments and flags
		if ctx.Args().Len() == 0 {
			first = mdFirst
			last = mdLast

			log.Noticef("Found first block (%v) and last block in AidaDb (%v)", first, last)
		} else if ctx.Args().Len() == 2 {
			// try to parse and check block range
			firstArg, lastArg, argErr := SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1), chainId)
			if argErr != nil {
				return nil, argErr
			}

			// find if values overlap
			first, last, err = adjustBlockRange(firstArg, lastArg, mdFirst, mdLast)
			if err != nil {
				return nil, err
			}
		}
	case BlockRangeArgsProfileDB:
		// process arguments and flags
		if ctx.Args().Len() == 3 {
			// try to extract block range from db metadata
			mdFirst, mdLast, err := getMdBlockRange(ctx)
			if err != nil {
				return nil, err
			}

			// try to parse and check block range
			firstArg, lastArg, argErr := SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1), chainId)
			if argErr != nil {
				return nil, argErr
			}

			// find if values overlap
			first, last, err = adjustBlockRange(firstArg, lastArg, mdFirst, mdLast)
			if err != nil {
				return nil, err
			}
			profileDB = ctx.Args().Get(2)
		} else if ctx.Args().Len() == 2 {
			return nil, fmt.Errorf("command requires profile db as argument")
		} else {
			return nil, fmt.Errorf("command requires 3 arguments")
		}
	case LastBlockArg:
		last, argErr = strconv.ParseUint(ctx.Args().Get(0), 10, 64)
		if argErr != nil {
			return nil, argErr
		}
	case NoArgs:
	default:
		return nil, errors.New("unknown mode; unable to process commandline arguments.")
	}

	cfg := createConfig(ctx)
	cfg.First = first
	cfg.Last = last
	cfg.ProfileDB = profileDB
	cfg.ChainID = chainId

	// --continue-on-failure implicitly enables transaction state validation
	validateTxState := ctx.Bool(ValidateFlag.Name) ||
		ctx.Bool(ValidateTxStateFlag.Name) ||
		ctx.Bool(ContinueOnFailureFlag.Name)
	cfg.ValidateTxState = validateTxState

	validateWorldState := ctx.Bool(ValidateFlag.Name) ||
		ctx.Bool(ValidateWorldStateFlag.Name)
	cfg.ValidateWorldState = validateWorldState

	setFirstBlockFromChainID(ChainID(cfg.ChainID))
	if cfg.RandomSeed < 0 {
		cfg.RandomSeed = int64(rand.Uint32())
	}
	err := setAidaDbRepositoryUrl(ChainID(cfg.ChainID))
	if err != nil {
		return cfg, fmt.Errorf("Unable to prepareUrl from ChainId %v; %v", cfg.ChainID, err)
	}

	if _, err := os.Stat(cfg.AidaDb); !os.IsNotExist(err) {
		log.Noticef("Found merged Aida-DB: %s redirecting UpdateDB, DeletedAccountDB, SubstateDB paths to it", cfg.AidaDb)
		cfg.UpdateDb = cfg.AidaDb
		cfg.DeletionDb = cfg.AidaDb
		cfg.SubstateDb = cfg.AidaDb
	}

	if ctx.String(ErigonBatchSizeFlag.Name) != "" {
		err := cfg.ErigonBatchSize.UnmarshalText([]byte(ctx.String(ErigonBatchSizeFlag.Name)))
		if err != nil {
			return cfg, fmt.Errorf("invalid batchSize provided: %v", err)
		}
	}

	if !cfg.Quiet {
		log.Noticef("Run config:")
		log.Infof("Block range: %v to %v", cfg.First, cfg.Last)
		if cfg.MaxNumTransactions >= 0 {
			log.Infof("Transaction limit: %d", cfg.MaxNumTransactions)
		}
		log.Infof("Chain id: %v (record & run-vm only)", cfg.ChainID)
		log.Infof("SyncPeriod length: %v", cfg.SyncPeriodLength)

		logDbMode := func(prefix, impl, variant string) {
			if cfg.DbImpl == "carmen" {
				log.Infof("%s: %v, DB variant: %v, DB schema: %d", prefix, impl, variant, cfg.CarmenSchema)
			} else {
				log.Infof("%s: %v, DB variant: %v", prefix, impl, variant)
			}
		}
		if !cfg.ShadowDb {
			logDbMode("Storage system", cfg.DbImpl, cfg.DbVariant)
		} else {
			logDbMode("Prime storage system", cfg.DbImpl, cfg.DbVariant)
			logDbMode("Shadow storage system", cfg.ShadowImpl, cfg.ShadowVariant)
		}
		log.Infof("Source storage directory (empty if new): %v", cfg.StateDbSrc)
		log.Infof("Working storage directory: %v", cfg.DbTmp)
		if cfg.ArchiveMode {
			log.Noticef("Archive mode: enabled")
			if cfg.ArchiveVariant == "" {
				log.Infof("Archive variant: <implementation-default>")
			} else {
				log.Infof("Archive variant: %s", cfg.ArchiveVariant)
			}
		} else {
			log.Infof("Archive mode: disabled")
		}
		log.Infof("Used VM implementation: %v", cfg.VmImpl)
		log.Infof("Update DB directory: %v", cfg.UpdateDb)
		if cfg.First == 0 {
			cfg.SkipPriming = true
		}
		if cfg.SkipPriming {
			log.Infof("Priming: Skipped")
		} else {
			log.Infof("Randomized Priming: %v", cfg.PrimeRandom)
			if cfg.PrimeRandom {
				log.Infof("Seed: %v, threshold: %v", cfg.RandomSeed, cfg.PrimeThreshold)
			}
			log.Infof("Update buffer size: %v bytes", cfg.UpdateBufferSize)
		}
		log.Infof("Validate world state: %v, validate tx state: %v", cfg.ValidateWorldState, cfg.ValidateTxState)
		log.Infof("Erigon batch size: %v", cfg.ErigonBatchSize.HumanReadable())
	}

	if cfg.ValidateTxState {
		log.Warning("Validation enabled, reducing Tx throughput")
	}
	if cfg.ShadowDb {
		log.Warning("DB shadowing enabled, reducing Tx throughput and increasing memory and storage usage")
	}
	if cfg.DbLogging {
		log.Warning("DB logging enabled, reducing Tx throughput")
	}
	if _, err := os.Stat(cfg.DeletionDb); os.IsNotExist(err) {
		log.Warning("Deleted-account-dir is not provided or does not exist")
		cfg.HasDeletedAccounts = false
	}
	if cfg.KeepDb && strings.Contains(cfg.DbVariant, "memory") {
		log.Warning("Unable to keep in-memory stateDB")
		cfg.KeepDb = false
	}
	if cfg.First != 0 && cfg.SkipPriming && cfg.ValidateWorldState {
		return cfg, fmt.Errorf("skipPriming and world-state validation can not be enabled at the same time")
	}

	return cfg, nil
}

// setAidaDbRepositoryUrl based on chain id selects correct aida-db repository url
func setAidaDbRepositoryUrl(chainId ChainID) error {
	if chainId == MainnetChainID {
		AidaDbRepositoryUrl = AidaDbRepositoryMainnetUrl
	} else if chainId == TestnetChainID {
		AidaDbRepositoryUrl = AidaDbRepositoryTestnetUrl
	} else {
		return fmt.Errorf("invalid chain id %d", chainId)
	}
	return nil
}

// SetBlockRange checks the validity of a block range and return the first and last block as numbers.
func SetBlockRange(firstArg string, lastArg string, chainId ChainID) (uint64, uint64, error) {
	var err error = nil
	first, ferr := strconv.ParseUint(firstArg, 10, 64)
	last, lerr := strconv.ParseUint(lastArg, 10, 64)

	if ferr != nil {
		first, err = setBlockNumber(firstArg, chainId)
		if err != nil {
			return 0, 0, err
		}
	}

	if lerr != nil {
		last, err = setBlockNumber(lastArg, chainId)
		if err != nil {
			return 0, 0, err
		}
	}

	if first > last {
		return 0, 0, fmt.Errorf("first block has larger number than last block")
	}

	return first, last, err
}

func setBlockNumber(arg string, chainId int) (uint64, error) {
	var blkNum uint64
	if chainId == TestnetChainID {
		if val, ok := hardForksTestnet[strings.ToLower(arg)]; ok {
			blkNum = val
		} else {
			return 0, fmt.Errorf("block number not a valid keyword or integer")
		}
	} else if chainId == MainnetChainID || chainId == UnknownChainID {
		if val, ok := hardForksMainnet[keyword]; ok {
			blkNum = val
		} else {
			return 0, fmt.Errorf("block number not a valid keyword or integer")
		}
	}

	// shift base block number by the offset
	if hasOffset {
		blkNum = offsetBlockNum(blkNum, symbol, offset)
	}

	return blkNum, nil
}

// parseOffset parse the hardfork keyword, offset value and a direction of the offset
func parseOffset(arg string) (string, string, uint64, error) {
	if strings.Contains(arg, "+") {
		if keyword, offset, ok := splitKeywordOffset(arg, "+"); ok {
			return strings.ToLower(keyword), "+", offset, nil
		}

		return "", "", 0, fmt.Errorf("block number not a valid keyword with offset")
	} else if strings.Contains(arg, "-") {
		if keyword, offset, ok := splitKeywordOffset(arg, "-"); ok {
			return strings.ToLower(keyword), "-", offset, nil
		}

		return "", "", 0, fmt.Errorf("block number not a valid keyword with offset")
	}

	return "", "", 0, fmt.Errorf("block number has invalid arithmetical sign")
}

// splitKeywordOffset split the hardfork keyword and the arithmetical sign determining the direction of the offset
func splitKeywordOffset(arg string, symbol string) (string, uint64, bool) {
	res := strings.Split(arg, symbol)

	if _, ok := hardForksMainnet[strings.ToLower(res[0])]; !ok {
		return "", 0, false
	}

	offset, err := strconv.ParseUint(res[1], 10, 64)
	if err != nil {
		return "", 0, false
	}

	return res[0], offset, true
}

// offsetBlockNum adds/subtracts the offset to/from block number
func offsetBlockNum(blkNum uint64, symbol string, offset uint64) uint64 {
	res := uint64(0)
	if symbol == "+" {
		res = blkNum + offset
	} else if symbol == "-" {
		res = blkNum - offset
	}

	return res
}

// getMdBlockRange gets block range from aidaDB metadata
func getMdBlockRange(ctx *cli.Context) (uint64, uint64, error) {
	aidaDb, err := rawdb.NewLevelDBDatabase(ctx.String(AidaDbFlag.Name), 1024, 100, "profiling", true)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, 0, fmt.Errorf("you either need to specify block range using arguments <first> <last>, or path to existing AidaDb (--%v) with block range in metadata", AidaDbFlag.Name)
		}
		return 0, 0, fmt.Errorf("cannot open aida-db; %v", err)
	}

	md := NewAidaDbMetadata(aidaDb, ctx.String(logger.LogLevelFlag.Name))
	mdFirst := md.GetFirstBlock()
	mdLast := md.GetLastBlock()

	if mdLast == 0 {
		return 0, 0, errors.New("your AidaDb does not have metadata with last block. Please run ./build/util-db info metadata --aida-db <path>")
	}

	err = aidaDb.Close()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot close db; %v", err)
	}

	return mdFirst, mdLast, nil
}

// adjustBlockRange finds overlap between metadata block range and block range specified by user in command line
func adjustBlockRange(firstArg uint64, lastArg uint64, mdFirst uint64, mdLast uint64) (uint64, uint64, error) {
	var first, last uint64

	if lastArg >= mdFirst && mdLast >= firstArg {
		// get first block number
		if firstArg > mdFirst {
			first = firstArg
		} else {
			first = mdFirst
		}

		// get last block number
		if lastArg < mdLast {
			last = lastArg
		} else {
			last = mdLast
		}

		return first, last, nil
	} else {
		return 0, 0, fmt.Errorf("given block range does NOT overlap with the block range of given aidaDB")
	}
}
