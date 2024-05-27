// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utildb

import (
	"encoding/binary"
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
	startingBlockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(startingBlockBytes, cfg.First)

	iter := aidaDb.NewIterator([]byte(substate.DestroyedAccountPrefix), startingBlockBytes)
	defer iter.Release()

	count := 0
	for iter.Next() {
		block, _, err := substate.DecodeDestroyedAccountKey(iter.Key())
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
