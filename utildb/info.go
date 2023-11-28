package utildb

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
)

// FindBlockRangeInUpdate finds the first and last block in the update set
func FindBlockRangeInUpdate(aidaDb ethdb.Database) (uint64, uint64, error) {
	udb := substate.NewUpdateDB(aidaDb)
	firstBlock, err := udb.GetFirstKey()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get first updateset; %v", err)
	}

	// get last updateset
	lastBlock, err := udb.GetLastKey()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get last updateset; %v", err)
	}
	return firstBlock, lastBlock, nil
}

// FindBlockRangeInDeleted finds the first and last block in the deleted accounts
func FindBlockRangeInDeleted(aidaDb ethdb.Database) (uint64, uint64, error) {
	ddb := substate.NewDestroyedAccountDB(aidaDb)
	firstBlock, err := ddb.GetFirstKey()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get first deleted accounts; %v", err)
	}

	// get last updateset
	lastBlock, err := ddb.GetLastKey()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get last deleted accounts; %v", err)
	}
	return firstBlock, lastBlock, nil
}

// FindBlockRangeInStateHash finds the first and last block in the state hash
func FindBlockRangeInStateHash(db ethdb.Database, log logger.Logger) (uint64, uint64, error) {
	var firstStateHashBlock, lastStateHashBlock uint64
	var err error
	firstStateHashBlock, err = utils.GetFirstStateHash(db)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get first state hash; %v", err)
	}

	lastStateHashBlock, err = utils.GetLastStateHash(db)
	if err != nil {
		log.Infof("Found first state hash at %v", firstStateHashBlock)
		return 0, 0, fmt.Errorf("cannot get last state hash; %v", err)
	}
	return firstStateHashBlock, lastStateHashBlock, nil
}

// GetSubstateCount in given AidaDb
func GetSubstateCount(cfg *utils.Config, aidaDb ethdb.Database) uint64 {
	substate.SetSubstateDbBackend(aidaDb)

	var count uint64

	iter := substate.NewSubstateIterator(cfg.First, 10)
	defer iter.Release()
	for iter.Next() {
		if iter.Value().Block > cfg.Last {
			break
		}
		count++
	}

	return count
}

// GetDeletedCount in given AidaDb
func GetDeletedCount(cfg *utils.Config, aidaDb ethdb.Database) (int, error) {
	db := substate.NewDestroyedAccountDB(aidaDb)

	accounts, err := db.GetAccountsDestroyedInRange(cfg.First, cfg.Last)
	if err != nil {
		return 0, fmt.Errorf("cannot Get all destroyed accounts; %v", err)
	}

	return len(accounts), nil
}

// GetUpdateCount in given AidaDb
func GetUpdateCount(cfg *utils.Config, aidaDb ethdb.Database) (uint64, error) {
	var count uint64

	start := substate.SubstateAllocKey(cfg.First)[2:]
	iter := aidaDb.NewIterator([]byte(substate.SubstateAllocPrefix), start)
	defer iter.Release()
	for iter.Next() {
		block, err := substate.DecodeUpdateSetKey(iter.Key())
		if err != nil {
			return 0, fmt.Errorf("cannot decode updateset key; %v", err)
		}
		if block > cfg.Last {
			break
		}
		count++
	}

	return count, nil
}

// GetStateHashCount in given AidaDb
func GetStateHashCount(cfg *utils.Config, aidaDb ethdb.Database) (uint64, error) {
	var count uint64

	hashProvider := utils.MakeStateHashProvider(aidaDb)
	for i := cfg.First; i <= cfg.Last; i++ {
		_, err := hashProvider.GetStateHash(int(i))
		if err != nil {
			if errors.Is(err, leveldb.ErrNotFound) {
				continue
			}
			return 0, err
		}
		count++
	}

	return count, nil
}
