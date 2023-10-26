package utils

//go:generate mockgen -source state_hash.go -destination state_hash_mocks.go -package utils

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rpc"
)

type StateHashProvider interface {
	GetStateHash(blockNumber int) (common.Hash, error)
}

func MakeStateHashProvider(db ethdb.Database) StateHashProvider {
	return &stateHashProvider{db}
}

type stateHashProvider struct {
	db ethdb.Database
}

func (p *stateHashProvider) GetStateHash(number int) (common.Hash, error) {
	hex := strconv.FormatUint(uint64(number), 16)
	stateRoot, err := p.db.Get([]byte(StateHashPrefix + "0x" + hex))
	if err != nil {
		return common.Hash{}, err
	}

	return common.Hash(stateRoot), nil
}

// StateHashScraper scrapes state hashes from a node and saves them to a leveldb database
func StateHashScraper(chainId ChainID, stateHashDb string, firstBlock, lastBlock int) error {
	db, err := rawdb.NewLevelDBDatabase(stateHashDb, 1024, 100, "state-hash", false)
	if err != nil {
		return fmt.Errorf("error opening stateHash leveldb %s: %v", stateHashDb, err)
	}
	defer db.Close()

	provider, err := GetProvider(chainId)
	if err != nil {
		return err
	}

	var client *rpc.Client
	client, err = rpc.Dial(provider)
	if err != nil {
		return fmt.Errorf("failed to connect to the rpc client: %v", err)
	}
	defer client.Close()

	var i = firstBlock

	// If firstBlock is 0, we need to get the state root for block 1 and save it as the state root for block 0
	// this is because the correct state root for block 0 is not available from the rpc node (at least in fantom mainnet)
	if firstBlock == 0 {
		block, err := retrieveStateRoot(client, "0x1")
		if err != nil {
			return err
		}

		err = saveStateRoot(db, block["stateRoot"].(string), "0x0")
		if err != nil {
			return err
		}
		i++
	}

	for ; i <= lastBlock; i++ {
		blockNumber := fmt.Sprintf("0x%x", i)
		block, err := retrieveStateRoot(client, blockNumber)
		if err != nil {
			return err
		}

		err = saveStateRoot(db, block["stateRoot"].(string), blockNumber)
		if err != nil {
			return err
		}

		if i%10000 == 0 {
			fmt.Printf("Block %d done!\n", i)
		}
	}

	return nil
}

// saveStateRoot saves the state root hash to the database
func saveStateRoot(db ethdb.Database, stateRoot string, blockNumber string) error {
	fullPrefix := StateHashPrefix + blockNumber
	err := db.Put([]byte(fullPrefix), []byte(stateRoot))
	if err != nil {
		return fmt.Errorf("unable to put state hash for block %s: %v", blockNumber, err)
	}
	return nil
}

// retrieveStateRoot gets the state root hash from the rpc node
func retrieveStateRoot(client *rpc.Client, blockNumber string) (map[string]interface{}, error) {
	var block map[string]interface{}
	err := client.Call(&block, "ftm_getBlockByNumber", blockNumber, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get block %s: %v", blockNumber, err)
	}
	return block, nil
}
