// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"context"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/opera"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/logger"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"sync"
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

	// load assembled accounts for the given root and write them into the snapshot database
	accounts, failed := LoadAccounts(context.Background(), opera.OpenStateDB(store), root, ctx.Int(flagWorkers), log)

	// find block information for the target state root hash
	dumpError := make(chan error, 1)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go lookupBlock(ctx.Context, store, outputDB, root, blockNumber, dumpError, wg, log)

	// wait for the writer to finish
	dbWriter(ctx.Context, outputDB, accounts, log)

	// any errors during the writer processing?
	for err = <-failed; err != nil; err = <-failed {
		log.Error(err.Error())
		dumpError <- err
	}
	close(dumpError)

	// wait for the block number to be written
	wg.Wait()

	log.Info("done")
	return nil
}

// lookupBlock searches for the block by root hash and stores the block once writer is done processing accounts.
func lookupBlock(ctx context.Context, store kvdb.Store, snap *snapshot.StateDB, root common.Hash, bn uint64, dumpError chan error, wg *sync.WaitGroup, log Logger) {
	defer wg.Done()

	block := make(chan uint64, 1)
	defer close(block)

	fail := make(chan error, 1)
	defer close(fail)

	// we already have it
	if bn > 0 {
		block <- bn
	} else {
		go func() {
			blk, err := opera.RootBLock(ctx, store, root)
			if err != nil {
				fail <- err
				return
			}
			block <- blk
		}()
	}

	ctxDone := ctx.Done()
	select {
	case <-ctxDone:
		return
	case err := <-fail:
		log.Errorf("failed to find block by root %s; %s", root.String(), err.Error())
		return
	case bn = <-block:
		log.Infof("found block %d for root state hash %s", bn, root.String())
	}

	select {
	case <-ctxDone:
		return
	case err := <-dumpError:
		if err != nil {
			break
		}

		log.Infof("saving block %d for root state hash %s", bn, root.String())
		err = snap.PutBlockNumber(bn)
		if err != nil {
			log.Errorf("could not store block number; %s", err.Error())
		}
	}
}
