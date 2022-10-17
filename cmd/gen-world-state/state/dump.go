// Package state implements executable entry points to the world state generator app.
package state

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/cmd/gen-world-state/flags"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/opera"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/snapshot"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
	"sync"
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
	accounts, readFailed := loadAccounts(ctx.Context, opera.OpenStateDB(store), root, ctx.Int(flags.Workers.Name), log)
	writeFailed := snapshot.NewQueueWriter(ctx.Context, outputDB, accounts)

	// find block information for the used state root hash
	block, lkpFailed := lookupBlock(ctx.Context, store, root, blockNumber)

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

// getChannelError checks for any pending error in a list of error channels.
// Please note this will block thread execution if any of the given channels is not closed yet.
func getChannelError(ec ...chan error) error {
	for _, e := range ec {
		err := <-e
		if err != nil {
			return err
		}
	}
	return nil
}

// lookupBlock searches for a block ID by root hash; if a block ID is already known, the search is skipped.
func lookupBlock(ctx context.Context, sourceDB kvdb.Store, root common.Hash, bn uint64) (chan uint64, chan error) {
	block := make(chan uint64, 1)
	fail := make(chan error, 1)

	// we may already have a block number from the committed state check;
	// if we do, just push it to output and cleanup channels
	if bn > 0 {
		block <- bn
		close(block)
		close(fail)

		return block, fail
	}

	// do a slow search in a separate thread
	go func() {
		defer close(block)
		defer close(fail)

		blk, err := opera.RootBLock(ctx, sourceDB, root)
		if err != nil {
			fail <- err
			return
		}

		block <- blk
	}()

	return block, fail
}

// loadAccounts iterates over EVM state trie at given root hash and sends assembled accounts into a channel.
// The provided output channel is closed when all accounts were sent.
func loadAccounts(ctx context.Context, db state.Database, root common.Hash, workers int, log *logging.Logger) (chan types.Account, chan error) {
	log.Infof("loading world state at root %s using %d worker threads", root.String(), workers)

	// we need to be able collect errors from all workers + raw account loader
	err := make(chan error, workers+1)
	out := make(chan types.Account, workers)

	go doLoadAccounts(ctx, db, root, out, err, workers, log)
	return out, err
}

// doLoadAccounts the state proxying account state assembly to workers.
func doLoadAccounts(ctx context.Context, db state.Database, root common.Hash, out chan types.Account, fail chan error, workers int, log *logging.Logger) {
	// signal issue between workers
	ca, cancel := context.WithCancel(ctx)

	// load account base data from the state DB into the workers input channel
	raw, rawErr := LoadRawAccounts(ca, db, root, log)
	defer func() {
		cancel()

		// do we have a raw accounts read error? copy the error to error output; this waits for the raw loader closing
		if err := <-rawErr; err != nil {
			fail <- err
		}
		close(fail)
		close(out)

		log.Debugf("account loader done")
	}()

	// monitor error channel and cancel context if an error is detected; this also closes the raw loader above
	go cancelCtxOnError(ca, cancel, fail, log)

	// start individual workers to extend accounts with code and storage and wait for them to finish
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go finaliseAccounts(ca, i, db, raw, out, fail, &wg, log)
	}
	wg.Wait()
}

// cancelCtxOnError monitors the error channel and cancels the context if an error is received.
func cancelCtxOnError(ctx context.Context, cancel context.CancelFunc, fail chan error, log *logging.Logger) {
	select {
	case <-ctx.Done():
		return
	case err, open := <-fail:
		cancel()
		if open {
			fail <- err // re-inject consumed error
			return
		}
		log.Errorf("worker error not propagated; %s", err.Error())
	}
}

// LoadRawAccounts iterates over EVM state and sends raw accounts to provided channel.
// The chanel is closed when all the available accounts were loaded.
// Raw accounts are not complete, e.g. contract storage and contract code is not loaded.
func LoadRawAccounts(ctx context.Context, db state.Database, root common.Hash, log *logging.Logger) (chan types.Account, chan error) {
	log.Debugf("loading accounts at root %s", root.String())

	assembly := make(chan types.Account, 25)
	err := make(chan error, 1)

	go loadRawAccounts(ctx, db, root, assembly, err, log)
	return assembly, err
}

// loadRawAccounts iterates over evm state then sends individual accounts to inAccounts channel
func loadRawAccounts(ctx context.Context, db state.Database, root common.Hash, raw chan types.Account, fail chan error, log *logging.Logger) {
	var count int64
	tick := time.NewTicker(5 * time.Second)
	defer func() {
		tick.Stop()
		close(raw)
		close(fail)
	}()

	// access trie
	stateTrie, err := db.OpenTrie(root)
	found := stateTrie != nil && err == nil
	if !found {
		fail <- fmt.Errorf("root hash %s not found", root.String())
		return
	}

	//  check existence of every code hash and rootHash of every storage trie
	ctxDone := ctx.Done()
	stateIt := stateTrie.NodeIterator(nil)
	for stateIt.Next(true) {
		if stateIt.Leaf() {
			count++
			addr := common.BytesToHash(stateIt.LeafKey())

			var acc state.Account
			if err = rlp.Decode(bytes.NewReader(stateIt.LeafBlob()), &acc); err != nil {
				fail <- fmt.Errorf("failed decoding account %s; %s\n", addr.String(), err.Error())
				return
			}

			select {
			case <-ctxDone:
				return
			case raw <- types.Account{Hash: addr, Account: acc}:
			}

			select {
			case <-tick.C:
				log.Infof("loaded %d accounts", count)
			default:
			}
		}
	}

	if stateIt.Error() != nil {
		fail <- fmt.Errorf("failed iterating trie at root %s; %s", root.String(), stateIt.Error())
	}

	log.Noticef("done %d accounts", count)
}

// finaliseAccounts worker finalizes accounts taken from an input queue by loading their code and storage;
// finalized accounts are send to output queue.
// This is done in parallel since some accounts with large storage space will take significant amount of time
// to be fully collected from the state trie.
func finaliseAccounts(ctx context.Context, wid int, db state.Database, in chan types.Account, out chan types.Account, fail chan error, wg *sync.WaitGroup, log *logging.Logger) {
	tick := time.NewTicker(20 * time.Second)
	defer func() {
		tick.Stop()
		wg.Done()

		log.Debugf("worker %d closed", wid)
	}()

	var last common.Hash
	var dur time.Duration
	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			return
		case <-tick.C:
			log.Debugf("worker #%d last account %s loaded in %s", wid, last.String(), dur.String())
		case acc, ok := <-in:
			if !ok {
				return
			}

			start := time.Now()
			err := assembleAccount(ctx, db, &acc)
			if err != nil {
				fail <- fmt.Errorf("failed assemling account %s; %s", acc.Hash.String(), err.Error())
				return
			}
			dur = time.Now().Sub(start)

			out <- acc
			last = acc.Hash
		}
	}
}

// assembleAccount finalises an account by adding contract code and storage, if any.
func assembleAccount(ctx context.Context, db state.Database, acc *types.Account) error {
	var err error

	// extract account code
	if !bytes.Equal(acc.CodeHash, types.EmptyCode) {
		codeHash := common.BytesToHash(acc.CodeHash)
		acc.Code, err = db.ContractCode(acc.Hash, codeHash)
		if err != nil {
			return fmt.Errorf("failed getting code %s at %s; %s", codeHash.String(), acc.Hash.String(), err.Error())
		}
	}

	// extract account storage
	if acc.Root != ethTypes.EmptyRootHash {
		acc.Storage, err = loadStorage(ctx, db, acc)
		if err != nil {
			return fmt.Errorf("failed loading storage %s at %s; %s\n", acc.Root.String(), acc.Hash.String(), err.Error())
		}
	}
	return nil
}

// loadStorage loads contract storage state by iterating over the storage trie and extracting key->value data.
func loadStorage(ctx context.Context, db state.Database, acc *types.Account) (map[common.Hash]common.Hash, error) {
	storage := map[common.Hash]common.Hash{}

	st, err := db.OpenStorageTrie(acc.Hash, acc.Root)
	if err != nil {
		return nil, fmt.Errorf("failed opening storage trie %s at %s; %s", acc.Root.String(), acc.Hash.String(), err.Error())
	}

	iter := st.NodeIterator(nil)
	ctxDone := ctx.Done()
	for iter.Next(true) {
		select {
		case <-ctxDone:
			return storage, ctx.Err()
		default:
		}

		if iter.Leaf() {
			key := common.BytesToHash(iter.LeafKey())
			value := iter.LeafBlob()

			if len(value) > 0 {
				_, content, _, err := rlp.Split(value)
				if err != nil {
					return nil, err
				}

				result := common.Hash{}
				result.SetBytes(content)
				storage[key] = result
			}
		}
	}

	if iter.Error() != nil {
		return nil, fmt.Errorf("failed iterating storage trie %s at %s; %s", acc.Root.String(), acc.Hash.String(), iter.Error().Error())
	}
	return storage, nil
}
