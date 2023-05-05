// Package state implements executable entry points to the world state generator app.
package state

import (
	"context"
	"fmt"
	"io"
	"time"

	substate "github.com/Fantom-foundation/Substate"

	eth_state "github.com/ethereum/go-ethereum/core/state"

	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/flags"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/world-state/db/opera"
	"github.com/Fantom-foundation/Aida/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// CmdDumpState defines a CLI command for dumping world state from a source database.
// We export all accounts (including contracts) with:
//   - Balance
//   - Nonce
//   - Code (hash + separate storage)
//   - Contract Storage
var CmdDumpState = cli.Command{
	Action:  DumpState,
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
		&utils.DbFlag,
		&utils.StateDbVariantFlag,
		&utils.SourceTableNameFlag,
		&utils.TrieRootHashFlag,
		&substate.WorkersFlag,
		&flags.TargetBlock,
		&utils.LogLevelFlag,
	},
}

// DumpState dumps state from given EVM trie into an output account-state database
func DumpState(ctx *cli.Context) error {
	// make config
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	// open the source trie DB
	store, err := opera.Connect(cfg.DbVariant, DefaultPath(ctx, &utils.DbFlag, ".opera/chaindata/leveldb-fsh"), cfg.SourceTableName)
	if err != nil {
		return err
	}
	defer opera.MustCloseStore(store)

	// try to open output DB
	outputDB, err := snapshot.OpenStateDB(cfg.WorldStateDb)
	if err != nil {
		return err
	}
	defer snapshot.MustCloseStateDB(outputDB)

	dumpCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	// make logger
	log := utils.NewLogger(cfg.LogLevel, "Dump")

	// blockNumber number to be stored in output db
	// root is root hash of storage at given block number
	blockNumber, root, err := blockNumberAndRoot(store, ctx.Uint64(flags.TargetBlock.Name), common.HexToHash(cfg.TrieRootHash), log)
	if err != nil {
		return err
	}

	// load assembled accounts for the given root and write them into the snapshot database
	workers := cfg.Workers
	accounts, readFailed := opera.LoadAccounts(dumpCtx, opera.OpenStateDB(store), root, workers)
	writeFailed := snapshot.NewQueueWriter(dumpCtx, outputDB, dumpProgressFactory(ctx.Context, accounts, workers, log))

	// find block information for the used state root hash
	block, lkpFailed := opera.GetBlockNumberByRoot(dumpCtx, store, root, blockNumber)

	// check for any error in above execution threads;
	// this will block until all threads above close their error channels
	err = getChannelError(readFailed, writeFailed, lkpFailed)
	if err != nil {
		endGracefully(cancel, readFailed, writeFailed, lkpFailed)
		return err
	}

	// write the block number into the database
	blockNumber = <-block
	err = outputDB.PutBlockNumber(blockNumber)
	if err != nil {
		return err
	}

	log.Noticef("importing addresses and storage keys")
	err = importHashesFromOpera(ctx, outputDB, opera.OpenStateDB(store), root, log)
	if err != nil {
		return err
	}

	// wait for all the threads to be done
	log.Noticef("block #%d done", blockNumber)

	return nil
}

// importHashesFromOpera extract addresses and storage keys from MPT dump and inserts into the worldstate
func importHashesFromOpera(ctx *cli.Context, db *snapshot.StateDB, store eth_state.Database, root common.Hash, log *logging.Logger) error {
	statedb, err := eth_state.NewWithSnapLayers(root, store, nil, 0)
	if err != nil {
		return fmt.Errorf("error calling opening StateDB; %v", err)
	}

	log.Noticef("dumping for addresses and storage keys")
	d := statedb.RawDump(nil)
	log.Noticef("dump finished")
	log.Noticef("loading keys into database")
	re := dumpWriter(d)
	return importCsv(ctx.App.Writer, re, db)
}

// dumpWriter iterates opera dump and inserts extracted hashes to the into the stream
func dumpWriter(d eth_state.Dump) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		for address, account := range d.Accounts {
			_, err := fmt.Fprint(pw, address.String()+"\n")
			if err != nil {
				panic(err)
			}
			for hash := range account.Storage {
				_, err = fmt.Fprint(pw, hash.String()+"\n")
				if err != nil {
					panic(err)
				}
			}
		}
		err := pw.Close()
		if err != nil {
			panic(err)
		}
	}()
	return pr
}

// endGracefully waits until all routines finish - preventing database to be closed prematurely
func endGracefully(cancel context.CancelFunc, ec ...<-chan error) {
	cancel()
	for i := 0; i < len(ec); i++ {
		for {
			_, ok := <-ec[i]
			if !ok {
				break
			}
		}
	}
}

// blockNumberAndRoot requires that root hash is determined.
// Also tries to load blockNumber, but it might still be 0 and required to be looked up outside of scope of this function.
func blockNumberAndRoot(store kvdb.Store, blockNumber uint64, root common.Hash, log *logging.Logger) (uint64, common.Hash, error) {
	var err error

	// neither blockNumber nor root hash were provided, try to use lastStateRoot containing latest block and root
	if root == snapshot.ZeroHash && blockNumber == 0 {
		// if the root has not been provided, try to use the latest
		root, blockNumber, err = opera.LatestStateRoot(store)
		if err != nil {
			log.Errorf("state root not found; %s", err.Error())
			return 0, snapshot.ZeroHash, err
		}

		log.Noticef("state root nor number were provided, using the latest block")
	}

	if root == snapshot.ZeroHash {
		// if only targetBlock was specified
		// look up root hash from block number
		root, err = opera.RootByBlockNumber(store, blockNumber)
		if err != nil {
			log.Errorf("unable to find root hash for block number %d; %s", blockNumber, err.Error())
			return 0, snapshot.ZeroHash, err
		}
	}

	if blockNumber != 0 {
		log.Noticef("state root %s at %d", root.String(), blockNumber)
	} else {
		log.Noticef("state root %s", root.String())
	}

	return blockNumber, root, nil
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
