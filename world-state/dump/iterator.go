// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	eth "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"sync"
)

// LoadAccounts iterates over EVM state trie at given root hash and sends assembled accounts into a channel.
// The provided output channel is closed when all accounts were sent.
func LoadAccounts(ctx context.Context, db state.Database, root common.Hash, workers int) (chan types.Account, chan error) {
	// we need to be able collect errors from all workers + raw account loader
	err := make(chan error, workers+1)
	out := make(chan types.Account, 25)

	go loadAccounts(ctx, db, root, out, err, workers)
	return out, err
}

// iterate the state proxying account state assembly to workers.
func loadAccounts(ctx context.Context, db state.Database, root common.Hash, out chan types.Account, fail chan error, workers int) {
	// signal issue between workers
	ca, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		close(out)
		close(fail)
	}()

	// load account base data from the state DB into the workers input channel
	raw, rawErr := LoadRawAccounts(ca, db, root)

	// monitor error channel and cancel context if an error is detected; this also closes the raw loader above
	go cancelCtxOnError(ca, cancel, fail)

	// start individual workers to extend accounts with code and storage and wait for them to finish
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go finaliseAccounts(ca, db, raw, out, fail, &wg)
	}
	wg.Wait()

	// do we have a raw accounts read error? copy the error to error output; this waits for the raw loader closing
	if err := <-rawErr; err != nil {
		fail <- err
	}
}

// cancelCtxOnError monitors the error channel and cancels the context if an error is received.
func cancelCtxOnError(ctx context.Context, cancel context.CancelFunc, fail chan error) {
	select {
	case <-ctx.Done():
		return
	case err := <-fail:
		cancel()
		fail <- err // re-feed consumed error
	}
}

// LoadRawAccounts iterates over EVM state and sends raw accounts to provided channel.
// The chanel is closed when all the available accounts were loaded.
// Raw accounts are not complete, e.g. contract storage and contract code is not loaded.
func LoadRawAccounts(ctx context.Context, db state.Database, root common.Hash) (chan types.Account, chan error) {
	assembly := make(chan types.Account, 25)
	err := make(chan error, 1)

	go loadRawAccounts(ctx, db, root, assembly, err)
	return assembly, err
}

// loadRawAccounts iterates over evm state then sends individual accounts to inAccounts channel
func loadRawAccounts(ctx context.Context, db state.Database, root common.Hash, raw chan types.Account, fail chan error) {
	defer func() {
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
	stateIt := stateTrie.NodeIterator(nil)
	for stateIt.Next(true) {
		if stateIt.Leaf() {
			addr := common.BytesToHash(stateIt.LeafKey())

			var acc state.Account
			if err = rlp.Decode(bytes.NewReader(stateIt.LeafBlob()), &acc); err != nil {
				fail <- fmt.Errorf("failed decoding account %s; %s\n", addr.String(), err.Error())
				return
			}

			select {
			case <-ctx.Done():
				return
			case raw <- types.Account{Hash: addr, Account: acc}:
			}
		}
	}

	if stateIt.Error() != nil {
		fail <- fmt.Errorf("failed iterating trie at root %s; %s", root.String(), stateIt.Error())
	}
}

// finaliseAccounts worker processes incomplete accounts from input queue and sends completed accounts to output.
func finaliseAccounts(ctx context.Context, db state.Database, in chan types.Account, out chan types.Account, fail chan error, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case acc, ok := <-in:
			if !ok {
				return
			}

			err := assembleAccount(ctx, db, &acc)
			if err != nil {
				fail <- fmt.Errorf("failed assemling account %s; %s", acc.Hash.String(), err.Error())
				return
			}

			out <- acc
		}
	}
}

// assembleAccount finalises an account by adding contract code and storage, if any.
func assembleAccount(ctx context.Context, db state.Database, acc *types.Account) error {
	var err error

	// extract account code
	codeHash := common.BytesToHash(acc.CodeHash)
	if codeHash != emptyCodeHash {
		acc.Code, err = db.ContractCode(acc.Hash, codeHash)
		if err != nil {
			return fmt.Errorf("failed getting code %s at %s; %s", codeHash.String(), acc.Hash.String(), err.Error())
		}
	}

	// extract account storage
	if acc.Root != eth.EmptyRootHash {
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
	for iter.Next(true) {
		select {
		case <-ctx.Done():
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
