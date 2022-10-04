package dump

import (
	"bytes"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/dump/kvdb2ethdb"
	accstate2 "github.com/Fantom-foundation/Aida-Testing/worldstate-cli/state-operation"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/leveldb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/nokeyiserr"
	"github.com/Fantom-foundation/lachesis-base/kvdb/pebble"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/urfave/cli/v2"
	logging "log"
	"sync"
)

const (
	InAccountsBufferSize  = 10
	OutAccountsBufferSize = 10
)

var (
	log       = logging.Default()
	emptyHash = common.Hash{}
	EmptyCode = crypto.Keccak256(
		nil,
	)
	emptyCodeHash = common.BytesToHash(EmptyCode)
)

// StateDumpAction dumps state of evm storage into account-state database
func StateDumpAction(ctx *cli.Context) error {
	cfg := parseArguments(ctx)

	// try to get DB producer
	db, err := getDBProducer(cfg)
	if err != nil {
		return err
	}

	// try to open DB producer
	kvdbDB, err := db.OpenDB(cfg.operaStateDBName)
	if err != nil {
		log.Println("Error while opening database: ", err)
		return err
	}
	defer func(kvdbDB kvdb.Store) {
		err = kvdbDB.Close()
		if err != nil {
			log.Println("Unable to close input database", "error", err)
		}
	}(kvdbDB)

	// evm data are stored under prefix M
	evmDB := table.New(kvdbDB, []byte(("M")))
	wrappedEvmDB := rawdb.NewDatabase(kvdb2ethdb.Wrap(nokeyiserr.Wrap(evmDB)))
	evmState := state.NewDatabaseWithConfig(wrappedEvmDB, &trie.Config{})

	// try to open output DB
	outputDB, err := accstate2.OpenOutputDB(cfg.outputDBDir)
	if err != nil {
		err = fmt.Errorf("error opening state-operation %s %s: %v", cfg.dbType, cfg.outputDBDir, err)
		log.Println(err)
		return err
	}
	defer func(Backend accstate2.BackendDatabase) {
		err = Backend.Close()
		if err != nil {
			log.Println("Unable to close output database", "error", err)
		}
	}(outputDB.Backend)

	// outAccounts channel is used to send prepared account data for writing to the output DB
	outAccounts := make(chan accstate2.Account, OutAccountsBufferSize)
	go stateIterator(evmState, cfg.rootHash, cfg.workers, outAccounts)

	dbWriter(outputDB, outAccounts)

	log.Println("Done.")
	return nil
}

// stateIterator iterates trough evmState at given rootHash
// starts workers number of threads, which are used for assembling account states
func stateIterator(evmState state.Database, rootHash common.Hash, workers int, outAccounts chan accstate2.Account) {
	// channel used for sending prepared accounts into workers
	inAccounts := make(chan accstate2.Account, InAccountsBufferSize)
	// main tree which feeds data trough inAccounts to workers
	go evmStateIterator(evmState, rootHash, inAccounts)

	var wg sync.WaitGroup

	// starting individual workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go handleAccounts(evmState, inAccounts, outAccounts, &wg)
	}

	// waiting until all workers finish their processing
	wg.Wait()
	close(outAccounts)
}

// handleAccounts listens on outAccounts channel for accounts then assembles all requested data
func handleAccounts(evmState state.Database, inAccounts chan accstate2.Account, outAccounts chan accstate2.Account, wg *sync.WaitGroup) {
	for {
		acc, ok := <-inAccounts
		if !ok {
			wg.Done()
			return
		}

		err := assembleAccount(evmState, &acc)
		if err != nil {
			log.Fatal("assembleAccount", "account", acc, "error", err)
		}
		outAccounts <- acc
	}
}

// dbWriter inserts received Accounts into database
func dbWriter(db *accstate2.StateDB, accounts chan accstate2.Account) {
	// listening to accounts channel until it gets closed
	for {
		acc, ok := <-accounts
		if !ok {
			return
		}

		// insert account code into database in separate record
		err := db.PutCode(acc.Code)
		if err != nil {
			log.Fatalf("record-replay: error putting code %s: %v", acc.CodeHash, err)
		}

		// insert account data
		err = db.PutAccount(acc)
		if err != nil {
			log.Fatal("Putting account into to database", "addrHash", acc.Hash, "error", err)
		}
	}
}

// getDBProducer loads from given data directory either leveldb or pebble database into producer
func getDBProducer(cfg *Config) (kvdb.IterableDBProducer, error) {
	var db kvdb.IterableDBProducer
	if cfg.dbType == "ldb" {
		// dbType is levelDB
		db = leveldb.NewProducer(cfg.operaStateDBDir, func(string) (int, int) {
			return 100 * opt.MiB, 1000
		})
	} else if cfg.dbType == "pbl" {
		// dbType is pebbleDB
		db = pebble.NewProducer(cfg.operaStateDBDir, func(string) (int, int) {
			return 100 * opt.MiB, 1000
		})
	}

	if db == nil {
		return nil, fmt.Errorf("failed to recognise inputdb type")
	}

	return db, nil
}

// evmStateIterator iterates over evm state then sends individual accounts to inAccounts channel
func evmStateIterator(evmState state.Database, rootHash common.Hash, inAccounts chan accstate2.Account) {
	stateTrie, err := evmState.OpenTrie(rootHash)
	found := stateTrie != nil && err == nil
	if !found {
		log.Fatal("Supplied root was not found in the database.")
	}
	log.Println("Starting trie iteration.")

	//  check existence of every code hash and rootHash of every storage trie
	stateIt := stateTrie.NodeIterator(nil)
	for stateIt.Next(true) {
		if stateIt.Leaf() {
			addrHash := common.BytesToHash(stateIt.LeafKey())

			var stateAcc state.Account
			if err = rlp.Decode(bytes.NewReader(stateIt.LeafBlob()), &stateAcc); err != nil {
				log.Fatalf("Failed to decode account as %s addr: %s", addrHash.String(), err.Error())
			}

			inAccounts <- accstate2.Account{
				Hash:    addrHash,
				Account: stateAcc,
			}
		}
	}

	if stateIt.Error() != nil {
		log.Fatalf("EVM state trie %s iteration error: %s", rootHash.String(), stateIt.Error())
	}

	// right after iteration is done closing inAccounts channel to let workers know that their work is finished
	close(inAccounts)
}

// assembleAccount assembles to account its code and storage
func assembleAccount(evmState state.Database, acc *accstate2.Account) error {
	var err error

	// extract account code
	codeHash := common.BytesToHash(acc.CodeHash)
	if codeHash != emptyCodeHash {
		acc.Code, err = evmState.ContractCode(acc.Hash, codeHash)
		if err != nil {
			log.Fatal("failed to get code %s at %s addr", codeHash.String(), acc.Hash.String())
			return err
		}
	}

	// extract account storage
	if acc.Root != types.EmptyRootHash {
		acc.Storage, err = contractStorage(acc, evmState, acc.Hash)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

// contractStorage assembles contract storage state map
func contractStorage(account *accstate2.Account, evmState state.Database, addrHash common.Hash) (map[common.Hash]common.Hash, error) {
	accStorageTmp := map[common.Hash]common.Hash{}
	storageTrie, storageErr := evmState.OpenStorageTrie(addrHash, account.Root)
	if storageErr != nil {
		err := fmt.Errorf("failed to open storage trie %s at %s addr: %s", account.Root.String(), addrHash.String(), storageErr.Error())
		return nil, err
	}

	storageIt := storageTrie.NodeIterator(nil)
	for storageIt.Next(true) {
		if storageIt.Leaf() {
			key := common.BytesToHash(storageIt.LeafKey())
			value := storageIt.LeafBlob()

			if len(value) > 0 {
				_, content, _, err := rlp.Split(value)
				if err != nil {
					return nil, err
				}
				result := common.Hash{}
				result.SetBytes(content)
				accStorageTmp[key] = result
			}
		}
	}

	if storageIt.Error() != nil {
		err := fmt.Errorf("EVM storage trie %s at %s addr iteration error: %s", account.Root.String(), addrHash.String(), storageIt.Error())
		return nil, err
	}
	return accStorageTmp, nil
}
