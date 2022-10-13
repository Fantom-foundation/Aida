// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"context"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/opera"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/logger"
	"github.com/ethereum/go-ethereum/common"
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
	store, err := opera.Connect(ctx.String(flagInputDBType), ctx.Path(flagInputDBPath), ctx.Path(flagStateDBName))
	if err != nil {
		return err
	}
	defer opera.MustCloseStore(store)

	// try to open output DB
	outputDB, err := snapshot.OpenStateDB(ctx.Path(FlagOutputDBPath))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(outputDB)

	// make logger
	log := logger.New(ctx.App.Writer, "info")

	// blockNumber number to be stored in output db
	var blockNumber uint64 = 0

	// load accounts from the given root
	// if the root has not been provided, try to use the latest
	root := common.HexToHash(ctx.String(flagStateRoot))
	if root == snapshot.ZeroHash {
		root, blockNumber, err = opera.LatestStateRoot(store)
		if err != nil {
			log.Errorf("state root not found; %s", err.Error())
			return err
		}

		log.Infof("state root not provided, using the latest %s", root.String())
	}

	var blockNumberChan chan uint64 = nil
	var blockNumberFailChan chan error = nil
	// root was not in blockEpochState, search for blockNumber of given root in block records
	if blockNumber == 0 {
		blockNumberChan, blockNumberFailChan = opera.RootBlock(store, root)
	}

	// load assembled accounts for the given root and write them into the snapshot database
	accounts, failed := LoadAccounts(context.Background(), opera.OpenStateDB(store), root, ctx.Int(flagWorkers), log)
	dbWriter(ctx.Context, outputDB, accounts)

	errorOccurred := false

	// any errors during the processing?
	for err = <-failed; err != nil; err = <-failed {
		errorOccurred = true
		log.Error(err.Error())
	}

	// no errors during processing write block number into database
	if !errorOccurred {
		storeBlockNumber(blockNumberChan, blockNumberFailChan, blockNumber, outputDB, log)
	}

	log.Info("done")
	return nil
}

// storeBlockNumber inserts block number to the output database
// blockNumber contains result only when root matched root of last block in database
func storeBlockNumber(blockNumberChan chan uint64, fail chan error, blockNumber uint64, outputDB *snapshot.StateDB, log Logger) {
	if fail != nil {
		err, ok := <-fail
		if err != nil && ok {
			log.Errorf("Error while getting block number from opera database occurred; %s", err.Error())
			return
		}
	}

	// blockNumberChan is only initialized when result isn't contained in blockNumber
	if blockNumberChan != nil {
		var ok bool
		blockNumber, ok = <-blockNumberChan
		if !ok {
			blockNumber = 0
		}
	}

	if blockNumber == 0 {
		log.Errorf("Block number for given root wasn't found in database")
		return
	}

	log.Infof("Inserting block number %d into database", blockNumber)
	err := outputDB.PutBlockNumber(blockNumber)
	if err != nil {
		log.Errorf("PutBlockNumber; %s", err.Error())
	}
}
