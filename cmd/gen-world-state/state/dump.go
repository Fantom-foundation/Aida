// Package state implements executable entry points to the world state generator app.
package state

import (
	"context"
	"github.com/Fantom-foundation/Aida/cmd/gen-world-state/flags"
	"github.com/Fantom-foundation/Aida/world-state/db/opera"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
	"time"
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
		&flags.SourceDBPath,
		&flags.SourceDBType,
		&flags.SourceTableName,
		&flags.TrieRootHash,
		&flags.Workers,
	},
}

// dumpState dumps state from given EVM trie into an output account-state database
func dumpState(ctx *cli.Context) error {
	// open the source trie DB
	store, err := opera.Connect(ctx.String(flags.SourceDBType.Name), DefaultPath(ctx, &flags.SourceDBPath, ".opera/chaindata/leveldb-fsh"), ctx.String(flags.SourceTableName.Name))
	if err != nil {
		return err
	}
	defer opera.MustCloseStore(store)

	// try to open output DB
	outputDB, err := snapshot.OpenStateDB(ctx.Path(flags.StateDBPath.Name))
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(outputDB)

	log := Logger(ctx, "dump")

	// blockNumber number to be stored in output db
	var blockNumber uint64 = 0

	// load accounts from the given root
	// if the root has not been provided, try to use the latest
	root := common.HexToHash(ctx.String(flags.TrieRootHash.Name))
	if root == snapshot.ZeroHash {
		root, blockNumber, err = opera.LatestStateRoot(store)
		if err != nil {
			log.Errorf("state root not found; %s", err.Error())
			return err
		}

		log.Noticef("state root not provided, using the latest %s", root.String())
	}

	// load assembled accounts for the given root and write them into the snapshot database
	workers := ctx.Int(flags.Workers.Name)
	accounts, readFailed := opera.LoadAccounts(ctx.Context, opera.OpenStateDB(store), root, workers)
	writeFailed := snapshot.NewQueueWriter(ctx.Context, outputDB, dumpProgressFactory(ctx.Context, accounts, workers, log))

	// find block information for the used state root hash
	block, lkpFailed := opera.GetBlockNumberByRoot(ctx.Context, store, root, blockNumber)

	// check for any error in above execution threads;
	// this will block until all threads above close their error channels
	err = getChannelError(readFailed, writeFailed, lkpFailed)
	if err != nil {
		return err
	}

	// write the block number into the database
	blockNumber = <-block
	err = outputDB.PutBlockNumber(blockNumber)
	if err != nil {
		return err
	}

	// wait for all the threads to be done
	log.Noticef("block #%d done", blockNumber)
	return nil
}

// dumpProgressFactory setups and executes dump progress logger.
func dumpProgressFactory(ctx context.Context, in <-chan types.Account, size int, log *logging.Logger) <-chan types.Account {
	out := make(chan types.Account, size)
	go dumpProgress(ctx, in, out, log)
	return out
}

// dumpProgress executes account channel proxying logging the progress by observing data passed.
func dumpProgress(ctx context.Context, in <-chan types.Account, out chan<- types.Account, log *logging.Logger) {
	var count int
	var last common.Hash
	tick := time.NewTicker(2 * time.Second)

	defer func() {
		tick.Stop()
		close(out)
	}()

	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			return
		case <-tick.C:
			log.Infof("processed %d accounts; visited %s", count, last.String())
		case acc, open := <-in:
			if !open {
				log.Noticef("%d accounts finished", count)
				return
			}

			out <- acc
			last = acc.Hash
			count++
		}
	}
}

// getChannelError checks for any pending error in a list of error channels.
// Please note this will block thread execution if any of the given channels is not closed yet.
func getChannelError(ec ...<-chan error) error {
	for _, e := range ec {
		err := <-e
		if err != nil {
			return err
		}
	}
	return nil
}
