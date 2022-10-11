package db

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/internal/db/kvdb2ethdb"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/internal/logger"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/leveldb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/nokeyiserr"
	"github.com/Fantom-foundation/lachesis-base/kvdb/pebble"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/Fantom-foundation/lachesis-base/utils/simplewlru"
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
	Usage:     "collect contents of mpt tree in opera database into world state database",
	ArgsUsage: "<rootHash>",
	Flags: []cli.Flag{
		&rootHashFlag,
		&dbDirFlag,
		&dbNameFlag,
		&dbTypeFlag,
		//substate.SubstateDirFlag,
	},
	Description: `
	"State exporter exports state db database from MPT tree at state of given root."`,
}

var (
	log       = logger.New()
	dbDatadir string
	dbName    string
	rootHash  common.Hash
	dbType    string
)

// stateDumpAction: dumps state of evm storage into substate database
func stateDumpAction(ctx *cli.Context) error {
	var err error

	parseArguments(ctx)

	var db = getDbProducer()
	if db == nil {
		err = errors.New("Failed to recognise db type")
		log.Error(err)
		return err
	}

	kvdbDb, err := db.OpenDB(dbName)
	if err != nil {
		log.Warning("Error while opening database: ", err)
		return err
	}
	defer kvdbDb.Close()

	//evm data are stored under prefix M
	evmDb := table.New(kvdbDb, []byte(("M")))

	wrappedEvmDb := rawdb.NewDatabase(
		kvdb2ethdb.Wrap(
			nokeyiserr.Wrap(
				evmDb)))

	evmState := state.NewDatabaseWithConfig(wrappedEvmDb, &trie.Config{})

	err = treeIteration(evmState, rootHash)

	return err
}

// getDbProducer loads from given datadir either leveldb or pebble database into producer
func getDbProducer() kvdb.IterableDBProducer {
	var db kvdb.IterableDBProducer
	if dbType == "ldb" {
		db = leveldb.NewProducer(dbDatadir, func(string) (int, int) {
			return 100 * opt.MiB, 1000
		})
	} else if dbType == "pbl" {
		db = pebble.NewProducer(dbDatadir, func(string) (int, int) {
			return 100 * opt.MiB, 1000
		})
	}
	return db
}

// parseArguments parse arguments
func parseArguments(ctx *cli.Context) {
	dbDatadir = ctx.Path(dbDirFlag.Name)
	dbName = ctx.String(dbNameFlag.Name)
	rootHash = common.HexToHash(ctx.String(rootHashFlag.Name))
	dbType = ctx.String(dbTypeFlag.Name)

	var emptyHash = common.Hash{}
	if rootHash == emptyHash {
		log.Critical("Root hash is not defined.")
	}
}

// treeIteration iterates trough evmState at given rootHash
func treeIteration(evmState state.Database, rootHash common.Hash) error {
	var (
		visitedHashes   = make([]common.Hash, 0, 1000000)
		visitedI        = 0
		checkedCache, _ = simplewlru.New(100000000, 100000000)
		cached          = func(h common.Hash) bool {
			_, ok := checkedCache.Get(h)
			return ok
		}
	)
	visited := func(h common.Hash, priority int) {
		base := 100000 * priority
		if visitedI%(1<<(len(visitedHashes)/base)) == 0 {
			visitedHashes = append(visitedHashes, h)
		}
		visitedI++
	}

	var (
		found         = false
		EmptyCode     = crypto.Keccak256(nil)
		emptyCodeHash = common.BytesToHash(EmptyCode)
		emptyHash     = common.Hash{}
	)

	stateTrie, err := evmState.OpenTrie(rootHash)
	found = stateTrie != nil && err == nil
	if !found {
		err = fmt.Errorf("Given root was not found in the database.")
		return err
	}
	log.Info("Starting trie iteration.")

	// check existence of every code hash and rootHash of every storage trie
	stateIt := stateTrie.NodeIterator(nil)
	for stateItSkip := false; stateIt.Next(!stateItSkip); {
		stateItSkip = false
		if stateIt.Hash() != emptyHash {
			if cached(stateIt.Hash()) {
				stateItSkip = true
				continue
			}
			visited(stateIt.Hash(), 2)
		}

		if stateIt.Leaf() {
			addrHash := common.BytesToHash(stateIt.LeafKey())

			var account state.Account
			if err = rlp.Decode(bytes.NewReader(stateIt.LeafBlob()), &account); err != nil {
				err = fmt.Errorf("Failed to decode accoun as %s addr: %s", addrHash.String(), err.Error())
				return err
			}

			codeHash := common.BytesToHash(account.CodeHash)
			if codeHash != emptyCodeHash && !cached(codeHash) {
				code, _ := evmState.ContractCode(addrHash, codeHash)
				if code == nil {
					err = fmt.Errorf("failed to get code %s at %s addr", codeHash.String(), addrHash.String())
					return err
				}
				checkedCache.Add(codeHash, true, 1)
			}

			if account.Root != types.EmptyRootHash && !cached(account.Root) {
				storageTrie, storageErr := evmState.OpenStorageTrie(addrHash, account.Root)
				if storageErr != nil {
					err = fmt.Errorf("failed to open storage trie %s at %s addr: %s", account.Root.String(), addrHash.String(), storageErr.Error())
					return err
				}
				storageIt := storageTrie.NodeIterator(nil)
				for storageItSkip := false; storageIt.Next(!storageItSkip); {
					storageItSkip = false
					if storageIt.Hash() != emptyHash {
						if cached(storageIt.Hash()) {
							storageItSkip = true
							continue
						}
						visited(storageIt.Hash(), 1)
					}
				}
				if storageIt.Error() != nil {
					err = fmt.Errorf("EVM storage trie %s at %s addr iteration error: %s", account.Root.String(), addrHash.String(), storageIt.Error())
					return err
				}
			}
		}
	}

	if stateIt.Error() != nil {
		err = fmt.Errorf("EVM state trie %s iteration error: %s", rootHash.String(), stateIt.Error())
		return err
	}
	for _, h := range visitedHashes {
		checkedCache.Add(h, true, 1)
	}
	visitedHashes = visitedHashes[:0]

	return nil
}
