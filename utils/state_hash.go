package utils

//go:generate mockgen -source state_hash.go -destination state_hash_mocks.go -package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/keycard-go/hexutils"
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
func StateHashScraper(ctx context.Context, chainId ChainID, operaPath string, db ethdb.Database, firstBlock, lastBlock uint64, log logger.Logger) error {
	ipcPath := operaPath + "/opera.ipc"

	client, err := getClient(ctx, chainId, ipcPath, log)
	if err != nil {
		return err
	}
	defer client.Close()

	var i = firstBlock

	// If firstBlock is 0, we need to get the state root for block 1 and save it as the state root for block 0
	// this is because the correct state root for block 0 is not available from the rpc node (at least in fantom mainnet and testnet)
	if firstBlock == 0 {
		block, err := retrieveStateRoot(client, "0x1")
		if err != nil {
			return err
		}

		if block == nil {
			return fmt.Errorf("block 1 not found")
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

		if block == nil {
			return fmt.Errorf("block %d not found", i)
		}

		err = saveStateRoot(db, block["stateRoot"].(string), blockNumber)
		if err != nil {
			return err
		}

		if i%10000 == 0 {
			log.Infof("Scraping block %d done!\n", i)
		}
	}

	return nil
}

// getClient returns a rpc/ipc client
func getClient(ctx context.Context, chainId ChainID, ipcPath string, log logger.Logger) (*rpc.Client, error) {
	var client *rpc.Client
	var err error

	_, errIpc := os.Stat(ipcPath)
	if errIpc == nil {
		// ipc file exists
		client, err = rpc.DialIPC(ctx, ipcPath)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to IPC at %s: %v", ipcPath, err)
		}
		log.Infof("Connected to IPC at %s", ipcPath)
		return client, err
	} else {
		var provider string
		provider, err = GetProvider(chainId)
		if err != nil {
			return nil, err
		}
		client, err = rpc.Dial(provider)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to the RPC client at %s: %v", provider, err)
		}
		log.Infof("Connected to RPC at %s", provider)
		return client, nil
	}
}

// saveStateRoot saves the state root hash to the database
func saveStateRoot(db ethdb.Database, stateRoot string, blockNumber string) error {
	fullPrefix := StateHashPrefix + blockNumber
	err := db.Put([]byte(fullPrefix), hexutils.HexToBytes(strings.TrimPrefix(stateRoot, "0x")))
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

// StateHashKeyToUint64 converts a state hash key to a uint64
func StateHashKeyToUint64(hexBytes []byte) (uint64, error) {
	prefix := []byte(StateHashPrefix)

	if len(hexBytes) >= len(prefix) && bytes.HasPrefix(hexBytes, prefix) {
		hexBytes = hexBytes[len(prefix):]
	}

	res, err := strconv.ParseUint(string(hexBytes), 0, 64)

	if err != nil {
		return 0, fmt.Errorf("cannot parse uint %v; %v", string(hexBytes), err)
	}
	return res, nil
}

// GetFirstStateHash returns the first block number for which we have a state hash
func GetFirstStateHash(db ethdb.Database) (uint64, error) {
	// TODO MATEJ will be fixed in future commit
	//iter := db.NewIterator([]byte(StateHashPrefix), []byte("0x"))
	//
	//defer iter.Release()
	//
	//// start with writing first block
	//if !iter.Next() {
	//	return 0, fmt.Errorf("no state hash found")
	//}
	//
	//firstStateHashBlock, err := StateHashKeyToUint64(iter.Key())
	//if err != nil {
	//	return 0, err
	//}
	//return firstStateHashBlock, nil
	return 0, fmt.Errorf("not implemented")
}

// GetLastStateHash returns the last block number for which we have a state hash
func GetLastStateHash(db ethdb.Database) (uint64, error) {
	// TODO MATEJ will be fixed in future commit
	//return GetLastKey(db, StateHashPrefix)
	return 0, fmt.Errorf("not implemented")
}
