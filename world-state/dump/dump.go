// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
	"log"
)

const (
	flagInputDBPath  = "input-db"
	flagInputDBType  = "input-db-type"
	flagStateDBName  = "input-db-name"
	flagOutputDBPath = "output-db"
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
	Action: dumpState,
	Name:   "dump",
	Usage:  "Extracts world state MPT trie at given root from input database into state snapshot output database.",
	Description: `
	The dump creates a snapshot of all accounts state (including contracts) exporting:
		- Balance
		- Nonce
		- Code (separate storage slot is used to store code data)
		- Contract Storage`,
	ArgsUsage: "<root> <input-db> <output-db> <input-db-name> <input-db-type> <workers>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  flagStateRoot,
			Usage: "Root hash of the state trie",
			Value: "",
		},
		&cli.PathFlag{
			Name:  flagInputDBPath,
			Usage: "Input state database path",
			Value: "",
		},
		&cli.PathFlag{
			Name:  flagOutputDBPath,
			Usage: "Output state snapshot database path",
			Value: "",
		},
		&cli.StringFlag{
			Name:  flagStateDBName,
			Usage: "Input state database name",
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
			Value: 25,
		},
	},
}

// dumpState dumps state from given EVM trie into an output account-state database
func dumpState(ctx *cli.Context) error {
	// open the source trie DB
	store, inputDB, err := db.OpenStateTrie(ctx.String(flagInputDBType), ctx.Path(flagInputDBPath), ctx.Path(flagStateDBName))
	if err != nil {
		return err
	}
	defer db.MustCloseStateTrie(store)

	// try to open output DB
	outputDB, err := db.OpenStateSnapshotDB(ctx.Path(flagOutputDBPath))
	if err != nil {
		return err
	}
	defer db.MustCloseSnapshotDB(outputDB)

	// load accounts from the given root
	root := common.HexToHash(ctx.String(flagInputDBType))
	workers := ctx.Int(ctx.String(flagWorkers))

	// what we do
	log.Printf("dumping state snapshot for root %s using %d workers\n", root.String(), workers)

	// load assembled accounts for the given root and write them into the snapshot database
	accounts, failed := LoadAccounts(inputDB, root, workers)
	dbWriter(outputDB, accounts)

	// any errors during the processing?
	for err = <-failed; err != nil; err = <-failed {
		log.Println(err.Error())
	}

	log.Println("done")
	return nil
}
