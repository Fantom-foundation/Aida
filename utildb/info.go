package utildb

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/syndtr/goleveldb/leveldb"
)

// FindBlockRangeInUpdate finds the first and last block in the update set
func FindBlockRangeInUpdate(udb db.UpdateDB) (uint64, uint64, error) {
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
func FindBlockRangeInDeleted(ddb *db.DestroyedAccountDB) (uint64, uint64, error) {
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
func FindBlockRangeInStateHash(db db.BaseDB, log logger.Logger) (uint64, uint64, error) {
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
func GetSubstateCount(cfg *utils.Config, sdb db.SubstateDB) uint64 {

	var count uint64

	iter := sdb.NewSubstateIterator(int(cfg.First), 10)
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
func GetDeletedCount(cfg *utils.Config, database db.BaseDB) (int, error) {
	startingBlockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(startingBlockBytes, cfg.First)

	iter := database.NewIterator([]byte(db.DestroyedAccountPrefix), startingBlockBytes)
	defer iter.Release()

	count := 0
	for iter.Next() {
		block, _, err := db.DecodeDestroyedAccountKey(iter.Key())
		if err != nil {
			return 0, fmt.Errorf("cannot Get all destroyed accounts; %v", err)
		}
		if block > cfg.Last {
			break
		}
		count++
	}

	return count, nil
}

// GetUpdateCount in given AidaDb
func GetUpdateCount(cfg *utils.Config, database db.BaseDB) (uint64, error) {
	var count uint64

	start := db.SubstateDBBlockPrefix(cfg.First)[2:]
	iter := database.NewIterator([]byte(db.SubstateDBPrefix), start)
	defer iter.Release()
	for iter.Next() {
		block, err := db.DecodeUpdateSetKey(iter.Key())
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
func GetStateHashCount(cfg *utils.Config, database db.BaseDB) (uint64, error) {
	var count uint64

	hashProvider := utils.MakeStateHashProvider(database)
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
