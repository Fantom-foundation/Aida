// Package opera implements Opera specific database interfaces for the world state manager.
package opera

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/Fantom-foundation/Aida/world-state/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// LoadAccounts iterates over EVM state trie at given root hash and sends assembled accounts into a channel.
// The provided output channel is closed when all accounts were sent.
func LoadAccounts(ctx context.Context, db state.Database, root common.Hash, workers int) (<-chan types.Account, <-chan error) {
	// we need to be able collect errors from all workers + raw account loader
	err := make(chan error, workers+1)
	out := make(chan types.Account, workers)

	go loadAccounts(ctx, db, root, out, err, workers)
	return out, err
}

// loadAccounts executes raw account loader along with specified number of assembly workers and coordinates their work.
func loadAccounts(ctx context.Context, db state.Database, root common.Hash, outAccount chan<- types.Account, fail chan<- error, workers int) {
	// in case a worker encounters an error, we will terminate the whole work context
	ca, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		close(fail)
		close(outAccount)
	}()

	// sync the workers closing
	var wg sync.WaitGroup
	rawAccount := make(chan types.Account, workers*2)
	workError := make(chan error, workers+1)

	// load account base data from the state DB into the workers input channel
	wg.Add(1)
	go loadRawAccounts(ca, db, root, rawAccount, workError, &wg)

	// start individual workers responsible for finalising accounts with separately loaded code and storage trie
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go finaliseAccount(ca, db, rawAccount, outAccount, workError, &wg)
	}

	// monitor internal error channel and cancel context if an error is detected
	go cancelCtxOnError(ca, cancel, workError, fail)
	wg.Wait()
}

// cancelCtxOnError monitors the error channel and cancels the context if an error is received.
func cancelCtxOnError(ctx context.Context, cancel context.CancelFunc, inError <-chan error, outError chan<- error) {
	select {
	case <-ctx.Done():
		return
	case err := <-inError:
		if err != nil {
			cancel()

			// inject the internal error to output error channel
			outError <- err
		}
		return
	}
}

// loadRawAccounts iterates over evm state then sends individual accounts to inAccounts channel
func loadRawAccounts(ctx context.Context, db state.Database, root common.Hash, raw chan<- types.Account, workError chan<- error, wg *sync.WaitGroup) {
	defer func() {
		close(raw)
		wg.Done()
	}()

	// access the state trie
	stateTrie, err := db.OpenTrie(root)
	found := stateTrie != nil && err == nil
	if !found {
		workError <- fmt.Errorf("root hash %s not found", root.String())
		return
	}

	//  check existence of every code hash and rootHash of every storage trie
	ctxDone := ctx.Done()
	stateIt := stateTrie.NodeIterator(nil)
	for stateIt.Next(true) {
		if stateIt.Leaf() {
			addr := common.BytesToHash(stateIt.LeafKey())

			var acc state.Account
			if err = rlp.Decode(bytes.NewReader(stateIt.LeafBlob()), &acc); err != nil {
				workError <- fmt.Errorf("failed to decode account %s; %s\n", addr.String(), err.Error())
				return
			}

			select {
			case <-ctxDone:
				return
			case raw <- types.Account{Hash: addr, Account: acc}:
			}
		}
	}

	if stateIt.Error() != nil {
		workError <- fmt.Errorf("failed to iterate trie at root %s; %s", root.String(), stateIt.Error())
	}
}

// finaliseAccount worker finalizes accounts taken from an input queue by loading their code and storage;
// finalized accounts are send to output queue. This is done in parallel across <workers> accounts.
func finaliseAccount(ctx context.Context, db state.Database, rawAccount <-chan types.Account, outAccount chan<- types.Account, workError chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			return
		case acc, open := <-rawAccount:
			// do we still expect more accounts to arrive?
			if !open {
				return
			}

			err := assembleAccount(ctx, db, &acc)
			if err != nil {
				workError <- fmt.Errorf("failed to assemble account %s; %s", acc.Hash.String(), err.Error())
				return
			}

			select {
			case <-ctxDone:
				return
			case outAccount <- acc:
			}
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
			return fmt.Errorf("failed to get contract code %s at %s; %s", codeHash.String(), acc.Hash.String(), err.Error())
		}
	}

	// extract account storage
	if acc.Root != ethTypes.EmptyRootHash {
		acc.Storage, err = loadStorage(ctx, db, acc)
		if err != nil {
			return fmt.Errorf("failed to load storage %s at %s; %s\n", acc.Root.String(), acc.Hash.String(), err.Error())
		}
	}
	return nil
}

// loadStorage loads contract storage state by iterating over the storage trie and extracting key->value data.
func loadStorage(ctx context.Context, db state.Database, acc *types.Account) (map[common.Hash]common.Hash, error) {
	storage := map[common.Hash]common.Hash{}

	st, err := db.OpenStorageTrie(acc.Hash, acc.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage trie %s at %s; %s", acc.Root.String(), acc.Hash.String(), err.Error())
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
		return nil, fmt.Errorf("failed to iterate storage trie %s at %s; %s", acc.Root.String(), acc.Hash.String(), iter.Error().Error())
	}
	return storage, nil
}
