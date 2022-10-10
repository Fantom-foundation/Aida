// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"context"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db"
	"github.com/Fantom-foundation/Aida-Testing/world-state/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
)

const (
	FlagOutputDBPath = "db"
	flagInputDBPath  = "input-db"
	flagInputDBType  = "input-db-type"
	flagStateDBName  = "input-db-name"
	flagStateRoot    = "root"
	flagWorkers      = "workers"
)

var (
	emptyCode     = crypto.Keccak256(nil)
	emptyCodeHash = common.BytesToHash(emptyCode)
)

// CmdDumpState defines a CLI command for dumping world state from a source database.
// We export all accounts (including contracts) with:
//   - Balance
//   - Nonce
//   - Code (hash + separate storage)
//   - Contract Storage
var CmdDumpState = cli.Command{
	Action:  dumpState,
	Name:    "dump",
	Aliases: []string{"d"},
	Usage:   "Extracts world state MPT trie at given root from input database into state snapshot output database.",
	Description: `The dump creates a snapshot of all accounts state (including contracts) exporting:
		- Balance
		- Nonce
		- Code (separate storage slot is used to store code data)
		- Contract Storage`,
	ArgsUsage: "<root> <input-db> <input-db-name> <input-db-type> <workers>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  flagStateRoot,
			Usage: "Root hash of the state trie",
			Value: "",
		},
		&cli.PathFlag{
			Name:     flagInputDBPath,
			Usage:    "Input database path with the state MPT",
			Value:    "",
			Required: true,
		},
		&cli.StringFlag{
			Name:  flagStateDBName,
			Usage: "Input state trie database table name",
			Value: "main",
		},
		&cli.StringFlag{
			Name:  flagInputDBType,
			Usage: "Type of input database (\"ldb\" or \"pbl\")",
			Value: "ldb",
		},
		&cli.IntFlag{
			Name:  flagWorkers,
			Usage: "Number of account processing threads",
			Value: 50,
		},
	},
}

// dumpState dumps state from given EVM trie into an output account-state database
func dumpState(ctx *cli.Context) error {
	// open the source trie DB
	store, err := db.Connect(ctx.String(flagInputDBType), ctx.Path(flagInputDBPath), ctx.Path(flagStateDBName))
	if err != nil {
		return err
	}
	defer db.MustCloseStore(store)

	// try to open output DB
	outputDB, err := db.OpenStateSnapshotDB(ctx.Path(FlagOutputDBPath))
	if err != nil {
		return err
	}
	defer db.MustCloseSnapshotDB(outputDB)

	// make logger
	log := logger.New(ctx.App.Writer, "info")

	// blockNumber number to be stored in output db
	var blockNumber uint64 = 0

	// load accounts from the given root
	// if the root has not been provided, try to use the latest
	root := common.HexToHash(ctx.String(flagStateRoot))
	if root == db.ZeroHash {
		root, blockNumber, err = db.LatestStateRoot(store)
		if err != nil {
			log.Errorf("state root not found; %s", err.Error())
			return err
		}

		log.Infof("state root not provided, using the latest %s", root.String())
	}

	var blockNumberChan chan uint64 = nil
	// root was not in BlockEpochState, search for blockNumber of given root in block records
	if blockNumber == 0 {
		blockNumberChan = RootBlock(store, root, log)
	}

	// load assembled accounts for the given root and write them into the snapshot database
	accounts, failed := LoadAccounts(context.Background(), db.OpenStateTrie(store), root, ctx.Int(flagWorkers), log)
	dbWriter(outputDB, accounts)

	errorOccurred := false

	// any errors during the processing?
	for err = <-failed; err != nil; err = <-failed {
		errorOccurred = true
		log.Error(err.Error())
	}

	// no errors during processing write block number into database
	if !errorOccurred {
		storeBlockNumber(blockNumberChan, blockNumber, outputDB, log)
	}

	log.Info("done")
	return nil
}
