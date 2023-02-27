// Package snapshot implements database interfaces for the world state manager.
package snapshot

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sync"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// SubstateIterator defines an interface of substate iterator we use to collect addresses.
type SubstateIterator interface {
	Next() bool
	Value() *substate.Transaction
	Release()
}

// ZeroAddressBytes represents the content of an empty address.
var ZeroAddressBytes = common.Address{}.Bytes()

// CollectAccounts collects accounts and storage hashes from the substate database iterator and sends them for processing.
func CollectAccounts(ctx context.Context, in SubstateIterator, toBlock uint64, workers int) (<-chan any, <-chan any) {
	outAddr := make(chan any, workers*10)
	outStorage := make(chan any, workers*10)
	go collectSubStateAccounts(ctx, in, toBlock, outAddr, outStorage, workers)
	return outAddr, outStorage
}

// WriteAccounts writes storage hashes and addresses received from an input queue into world state snapshot database.
func WriteAccounts(ctx context.Context, in <-chan any, db *StateDB) chan error {
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		var hashing = crypto.NewKeccakState()

		ctxDone := ctx.Done()
		for {
			select {
			case <-ctxDone:
				errChan <- ctx.Err()
			case data, open := <-in:
				if !open {
					return
				}

				var err error
				switch d := data.(type) {
				case common.Address:
					// calculate the hash of the account
					err = db.PutHashToAccountAddress(crypto.HashData(hashing, d.Bytes()), d)
				case common.Hash:
					// calculate the hash of the account
					err = db.PutHashToStorage(crypto.HashData(hashing, d.Bytes()), d)
				default:
					err = fmt.Errorf("unexpected type while writting to database %s", reflect.TypeOf(data))
					return
				}

				if err != nil {
					errChan <- err
					return
				}
			}
		}
	}()
	return errChan
}

// FilterUnique extracts storage hashes and addresses from input queue and sends only unique occurrences to the output.
// The filter will close output channel once done processing incoming storage hashes.
func FilterUnique(ctx context.Context, in <-chan any, out chan<- any) {
	defer close(out)

	visited := make(map[string]bool, 20_000_000)
	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			return
		case data, open := <-in:
			if !open {
				return
			}

			var hex string
			switch d := data.(type) {
			case common.Address:
				hex = d.Hex()
			case common.Hash:
				hex = d.Hex()
			default:
				//	TODO deal with error
			}

			// is this a unique address?
			_, found := visited[hex]
			if found {
				continue
			}

			select {
			case <-ctxDone:
				return
			case out <- data:
				visited[hex] = true
			}
		}
	}
}

// collectSubStateAccounts iterates SubState database transactions and sends them for processing to a worker channel.
// The iteration walker will close output channel once all the internal workers are done collecting addresses.
func collectSubStateAccounts(ctx context.Context, in SubstateIterator, toBlock uint64, outAddr chan<- any, outStorage chan<- any, workers int) {
	defer close(outAddr)
	defer close(outStorage)

	// prepare structures for account collectors
	work := make(chan *substate.Transaction, workers)
	var wg sync.WaitGroup

	// start account collector workers
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go extractSubStateAccounts(ctx, work, outAddr, outStorage, &wg)
	}

	// iterate SubStates to get transactions
	ctxDone := ctx.Done()
	for in.Next() {
		tx := in.Value()
		if toBlock > 0 && tx.Block > toBlock {
			break
		}

		select {
		case <-ctxDone:
			break
		case work <- tx:
		}
	}

	// signal workers we are done iterating and wait for their termination
	close(work)
	wg.Wait()
}

// extractSubStateAccounts worker executes account and storage collector job on substate records received via input queue.
// Found account addresses are sent to an output queue for processing.
// We do not care about sending unique address from the worker since it's expected to run in parallel,
// the filtering should be done later down the queue line.
func extractSubStateAccounts(ctx context.Context, in <-chan *substate.Transaction, outAddr chan<- any, outStorage chan<- any, wg *sync.WaitGroup) {
	defer wg.Done()

	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			return
		case tx, open := <-in:
			if !open {
				return
			}

			// non-zero env coinbase
			if tx.Substate.Env != nil && !bytes.Equal(tx.Substate.Env.Coinbase.Bytes(), ZeroAddressBytes) {
				outAddr <- tx.Substate.Env.Coinbase
			}

			// message sender and recipient
			if tx.Substate.Message != nil {
				outAddr <- tx.Substate.Message.From

				if tx.Substate.Message.To != nil {
					outAddr <- *tx.Substate.Message.To
				}
			}

			// input alloc
			if tx.Substate.InputAlloc != nil {
				for adr, state := range tx.Substate.InputAlloc {
					outAddr <- adr

					for hash := range state.Storage {
						outStorage <- hash
					}
				}
			}

			// output alloc
			if tx.Substate.OutputAlloc != nil {
				for adr, state := range tx.Substate.OutputAlloc {
					outAddr <- adr

					for hash := range state.Storage {
						outStorage <- hash
					}
				}
			}

			// non-zero result contract address
			if tx.Substate.Result != nil && !bytes.Equal(tx.Substate.Result.ContractAddress.Bytes(), ZeroAddressBytes) {
				outAddr <- tx.Substate.Result.ContractAddress
			}
		}
	}
}
