package dump

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/internal/accstate"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/internal/dump/kvdb2ethdb"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/internal/logger"
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
	"sync"
)

// StateDumpCommand command
var StateDumpCommand = cli.Command{
	Action:    stateDumpAction,
	Name:      "state-dump",
	Usage:     "collect contents of mpt tree from opera database into world state database",
	ArgsUsage: "<root-hash> <state-dump-dir> <accstate-dump-dir> <dump-name> <state-dump-type> <workers>",
	Flags: []cli.Flag{
		&rootHashFlag,
		&stateDBFlag,
		&outputDBFlag,
		&dbNameFlag,
		&dbTypeFlag,
		&workersFlag,
	},
	Description: `
	"State dump dumps evm storage database from MPT tree at state of given root."`,
}

// Config represents parsed arguments
type Config struct {
	operaStateDBDir  string
	outputDBDir      string
	operaStateDBName string
	rootHash         common.Hash
	dbType           string
	workers          int
}

const (
	InAccountsBufferSize  = 10
	OutAccountsBufferSize = 10
)

var (
	log           = logger.New()
	emptyHash     = common.Hash{}
	EmptyCode     = crypto.Keccak256(nil)
	emptyCodeHash = common.BytesToHash(EmptyCode)
)

// stateDumpAction: dumps state of evm storage into account-state database
func stateDumpAction(ctx *cli.Context) error {
	cfg := parseArguments(ctx)

	// try to get DB producer
	db, err := getDBProducer(cfg)
	if err != nil {
		return err
	}

	// try to open DB producer
	kvdbDB, err := db.OpenDB(cfg.operaStateDBName)
	if err != nil {
		log.Warning("Error while opening database: ", err)
		return err
	}
	defer kvdbDB.Close()

	// evm data are stored under prefix M
	evmDB := table.New(kvdbDB, []byte(("M")))
	wrappedEvmDB := rawdb.NewDatabase(kvdb2ethdb.Wrap(nokeyiserr.Wrap(evmDB)))
	evmState := state.NewDatabaseWithConfig(wrappedEvmDB, &trie.Config{})

	// try to open output DB
	outputDB, err := accstate.OpenOutputDB(cfg.outputDBDir)
	if err != nil {
		err = errors.New(fmt.Sprintf("error opening accstate %s %s: %v", cfg.dbType, cfg.outputDBDir, err))
		log.Warning(err)
		return err
	}
	defer outputDB.Backend.Close()

	// outAccounts channel is used to send prepared account data for writing to the output DB
	outAccounts := make(chan accstate.Account, OutAccountsBufferSize)
	go stateIterator(evmState, cfg.rootHash, cfg.workers, outAccounts)

	dbWriter(outputDB, outAccounts)

	log.Info("Done.")
	return nil
}

// stateIterator iterates trough evmState at given rootHash
// starts workers number of threads, which are used for assembling account states
func stateIterator(evmState state.Database, rootHash common.Hash, workers int, outAccounts chan accstate.Account) {
	// channel used for sending prepared accounts into workers
	inAccounts := make(chan accstate.Account, InAccountsBufferSize)
	// main tree which feeds data trough inAccounts to workers
	go evmStateIterator(evmState, rootHash, inAccounts)

	var wg sync.WaitGroup

	// starting individual workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleAccounts(evmState, inAccounts, outAccounts)
		}()
	}

	// waiting until all workers finish their processing
	wg.Wait()
	close(outAccounts)
}

// handleAccounts listens on outAccounts channel for accounts then assembles all requested data
func handleAccounts(evmState state.Database, inAccounts chan accstate.Account, outAccounts chan accstate.Account) {
	for {
		acc, ok := <-inAccounts
		if !ok {
			return
		}
		assembleAccount(evmState, acc, outAccounts)
	}
}

// dbWriter inserts received Accounts into database
func dbWriter(db *accstate.StateDB, accounts chan accstate.Account) {
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
		return nil, errors.New("failed to recognise dump type")
	}

	return db, nil
}

// parseArguments parse arguments into Config
func parseArguments(ctx *cli.Context) *Config {
	// check whether supplied rootHash is not empty
	rootHash := common.HexToHash(ctx.String(rootHashFlag.Name))
	if rootHash == emptyHash {
		log.Critical("Root hash is not defined.")
	}

	return &Config{
		rootHash:         rootHash,
		operaStateDBDir:  ctx.Path(stateDBFlag.Name),
		outputDBDir:      ctx.Path(outputDBFlag.Name),
		operaStateDBName: ctx.String(dbNameFlag.Name),
		dbType:           ctx.String(dbTypeFlag.Name),
		workers:          ctx.Int(workersFlag.Name)}
}

// evmStateIterator iterates over evm state then sends individual accounts to inAccounts channel
func evmStateIterator(evmState state.Database, rootHash common.Hash, inAccounts chan accstate.Account) {
	stateTrie, err := evmState.OpenTrie(rootHash)
	found := stateTrie != nil && err == nil
	if !found {
		log.Fatal("Supplied root was not found in the database.")
	}
	log.Info("Starting trie iteration.")

	//  check existence of every code hash and rootHash of every storage trie
	stateIt := stateTrie.NodeIterator(nil)
	for stateIt.Next(true) {
		if stateIt.Leaf() {
			addrHash := common.BytesToHash(stateIt.LeafKey())

			var stateAcc state.Account
			if err = rlp.Decode(bytes.NewReader(stateIt.LeafBlob()), &stateAcc); err != nil {
				log.Fatalf("Failed to decode account as %s addr: %s", addrHash.String(), err.Error())
			}

			inAccounts <- accstate.Account{
				Hash:     addrHash,
				Root:     stateAcc.Root,
				CodeHash: stateAcc.CodeHash,
				Nonce:    stateAcc.Nonce,
				Balance:  stateAcc.Balance,
			}
		}
	}

	if stateIt.Error() != nil {
		log.Fatalf("EVM state trie %s iteration error: %s", rootHash.String(), stateIt.Error())
	}

	close(inAccounts)
}

// assembleAccount assembles Account data
// then sends them trough outAccounts to be written in database
func assembleAccount(evmState state.Database, acc accstate.Account, outAccounts chan accstate.Account) {
	var err error

	// extract account code
	codeHash := common.BytesToHash(acc.CodeHash)
	if codeHash != emptyCodeHash {
		code, _ := evmState.ContractCode(acc.Hash, codeHash)
		if code == nil {
			log.Fatal("failed to get code %s at %s addr", codeHash.String(), acc.Hash.String())
		}
		acc.Code = code
	}

	// extract account storage
	if acc.Root != types.EmptyRootHash {
		acc.Storage, err = iterateTroughAccountStorage(acc, evmState, acc.Hash)
		if err != nil {
			log.Fatal(err)
		}
	}
	outAccounts <- acc
}

// iterateTroughAccountStorage opens storage trie for specific account
func iterateTroughAccountStorage(account accstate.Account, evmState state.Database, addrHash common.Hash) (map[common.Hash]common.Hash, error) {
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
