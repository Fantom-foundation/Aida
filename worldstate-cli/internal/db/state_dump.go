package db

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/internal/db/kvdb2ethdb"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/internal/logger"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/internal/substate"
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
)

// StateDumpCommand command
var StateDumpCommand = cli.Command{
	Action:    stateDumpAction,
	Name:      "state-dump",
	Usage:     "collect contents of mpt tree from opera database into world state database",
	ArgsUsage: "<rootHash>",
	Flags: []cli.Flag{
		&rootHashFlag,
		&stateDbDirFlag,
		&substateDirFlag,
		&dbNameFlag,
		&dbTypeFlag,
		&workersFlag,
	},
	Description: `
	"State dump dumps evm storage database from MPT tree at state of given root."`,
}

type Config struct {
	operastateDatadir string
	substateDatadir   string
	dbName            string
	rootHash          common.Hash
	dbType            string
	workers           int
}

type SubstateAccountWrapper struct {
	addressHash     common.Hash
	substateAccount substate.SubstateAccount
}

type accountDataWrapper struct {
	addrHash common.Hash
	account  state.Account
}

var (
	log           = logger.New()
	emptyHash     = common.Hash{}
	EmptyCode     = crypto.Keccak256(nil)
	emptyCodeHash = common.BytesToHash(EmptyCode)
)

// stateDumpAction: dumps state of evm storage into substate database
func stateDumpAction(ctx *cli.Context) error {
	cfg := parseArguments(ctx)

	db, err := getDbProducer(cfg)
	if err != nil {
		return err
	}

	kvdbDb, err := db.OpenDB(cfg.dbName)
	if err != nil {
		log.Warning("Error while opening database: ", err)
		return err
	}
	defer kvdbDb.Close()

	//evm data are stored under prefix M
	evmDb := table.New(kvdbDb, []byte(("M")))
	wrappedEvmDb := rawdb.NewDatabase(kvdb2ethdb.Wrap(nokeyiserr.Wrap(evmDb)))
	evmState := state.NewDatabaseWithConfig(wrappedEvmDb, &trie.Config{})

	substateDb := substate.OpenSubstateDB(cfg.substateDatadir)
	defer substateDb.Backend.Close()

	finishedSubstateAccountsQueue := make(chan SubstateAccountWrapper, 100)
	defer close(finishedSubstateAccountsQueue)

	stopSignal := make(chan bool, 0)
	defer close(stopSignal)
	go dbFlusher(finishedSubstateAccountsQueue, stopSignal, substateDb)

	stateIterator(evmState, cfg.rootHash, cfg.workers, finishedSubstateAccountsQueue)

	log.Info("Finishing.")
	stopSignal <- true
	return nil
}

// stateIterator iterates trough evmState at given rootHash
// starts workers number of threads, which are used for assembling substate accounts
func stateIterator(evmState state.Database, rootHash common.Hash, workers int, finishedSubstateAccountsQueue chan SubstateAccountWrapper) {
	preparedAccountsQueue := make(chan accountDataWrapper, 0)
	defer close(preparedAccountsQueue)
	stopSignal := make(chan bool, 0)
	defer close(stopSignal)

	//starting individual workers
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-stopSignal:
					{
						return
					}
				case data := <-preparedAccountsQueue:
					{
						assembleSubstateAccount(evmState, data.account, data.addrHash, finishedSubstateAccountsQueue)
					}
				}
			}
		}()
	}

	//main tree which feeds data trough preparedAccountsQueue to workers
	err := evmStateIterator(evmState, rootHash, preparedAccountsQueue)
	if err != nil {
		log.Fatal("evmStateIterator", "error", err)
	}

	//closing workers
	for i := 0; i < workers; i++ {
		stopSignal <- true
	}
}

// dbFlusher inserts received  substateAccounts into database
func dbFlusher(foundAccountsQueue chan SubstateAccountWrapper, stopSignal chan bool, db *substate.SubstateDB) {
	for {
		select {
		case <-stopSignal:
			{
				return
			}
		case data := <-foundAccountsQueue:
			{
				substateAccountBytes, err := rlp.EncodeToBytes(data.substateAccount.NewSubstateAccountRLP())
				if err != nil {
					log.Fatal("Encoding data to rlp", "addrHash", data.addressHash, "error", err)
				}

				err = db.Backend.Put(data.addressHash.Bytes(), substateAccountBytes)
				if err != nil {
					log.Fatal("Putting data to database", "addrHash", data.addressHash, "error", err)
				}
			}
		}
	}
}

// getDbProducer loads from given datadir either leveldb or pebble database into producer
func getDbProducer(cfg *Config) (kvdb.IterableDBProducer, error) {
	var db kvdb.IterableDBProducer
	if cfg.dbType == "ldb" {
		db = leveldb.NewProducer(cfg.operastateDatadir, func(string) (int, int) {
			return 100 * opt.MiB, 1000
		})
	} else if cfg.dbType == "pbl" {
		db = pebble.NewProducer(cfg.operastateDatadir, func(string) (int, int) {
			return 100 * opt.MiB, 1000
		})
	}

	if db == nil {
		return nil, errors.New("Failed to recognise db type.")
	}

	return db, nil
}

// parseArguments parse arguments
func parseArguments(ctx *cli.Context) *Config {
	rootHash := common.HexToHash(ctx.String(rootHashFlag.Name))
	if rootHash == emptyHash {
		log.Critical("Root hash is not defined.")
	}

	return &Config{
		operastateDatadir: ctx.Path(stateDbDirFlag.Name),
		substateDatadir:   ctx.Path(substateDirFlag.Name),
		dbName:            ctx.String(dbNameFlag.Name),
		rootHash:          rootHash,
		dbType:            ctx.String(dbTypeFlag.Name),
		workers:           ctx.Int(workersFlag.Name)}
}

// evmStateIterator iterates trough evmState at given rootHash
func evmStateIterator(evmState state.Database, rootHash common.Hash, workerLoadChan chan accountDataWrapper) error {
	stateTrie, err := evmState.OpenTrie(rootHash)
	found := stateTrie != nil && err == nil
	if !found {
		err = fmt.Errorf("Supplied root was not found in the database.")
		return err
	}
	log.Info("Starting trie iteration.")

	// check existence of every code hash and rootHash of every storage trie
	stateIt := stateTrie.NodeIterator(nil)
	for stateIt.Next(true) {
		if stateIt.Leaf() {
			addrHash := common.BytesToHash(stateIt.LeafKey())

			var account state.Account
			if err = rlp.Decode(bytes.NewReader(stateIt.LeafBlob()), &account); err != nil {
				err = fmt.Errorf("Failed to decode accoun as %s addr: %s", addrHash.String(), err.Error())
				return err
			}

			workerLoadChan <- accountDataWrapper{
				addrHash,
				account,
			}
		}
	}

	if stateIt.Error() != nil {
		err = fmt.Errorf("EVM state trie %s iteration error: %s", rootHash.String(), stateIt.Error())
		return err
	}

	return nil
}

// assembleSubstateAccount completes all SubstateAcccount data
// then sends them trough finishedSubstateAccountsQueue to be written in database
func assembleSubstateAccount(evmState state.Database, account state.Account, addrHash common.Hash, finishedSubstateAccountsQueue chan SubstateAccountWrapper) {
	var err error
	var acc = substate.SubstateAccount{
		Nonce:   account.Nonce,
		Balance: account.Balance,
	}

	codeHash := common.BytesToHash(account.CodeHash)
	if codeHash != emptyCodeHash {
		code, _ := evmState.ContractCode(addrHash, codeHash)
		if code == nil {
			err = fmt.Errorf("failed to get code %s at %s addr", codeHash.String(), addrHash.String())
			log.Fatal(err)
		}
		acc.Code = code
	}

	if account.Root != types.EmptyRootHash {
		acc.Storage, err = iterateTroughAccountStorage(account, evmState, addrHash)
		if err != nil {
			log.Fatal(err)
		}
	}
	finishedSubstateAccountsQueue <- SubstateAccountWrapper{addressHash: addrHash, substateAccount: acc}
}

// iterateTroughAccountStorage opens storage trie for specific account
func iterateTroughAccountStorage(account state.Account, evmState state.Database, addrHash common.Hash) (map[common.Hash]common.Hash, error) {
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
