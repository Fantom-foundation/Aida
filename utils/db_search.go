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

package utils

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"
)

// TODO MERGE IN FUTURE - this file has almost same functionality as getLastSubstateKey in static_substate_db.go
// either should be generalised there or functionality could be moved to separate library and used in both places

type SearchableDB struct {
	ethdb.Database
}

func NewSearchableDB(backend ethdb.Database) *SearchableDB {
	return &SearchableDB{backend}
}

func GetLastKey(dbIn ethdb.Database, keyPrefix string) (uint64, error) {
	db := NewSearchableDB(dbIn)

	zeroBytes, err := db.getLongestEncodedKeyZeroPrefixLength(keyPrefix)
	if err != nil {
		return 0, err
	}
	var lastKeyPrefix []byte
	if zeroBytes > 0 {
		blockBytes := make([]byte, zeroBytes)

		lastKeyPrefix = append([]byte(keyPrefix), blockBytes...)
	} else {
		lastKeyPrefix = []byte(keyPrefix)
	}

	stateHashPrefixSize := len([]byte(keyPrefix))

	// binary search for biggest key
	for {
		nextBiggestPrefixValue, err := db.binarySearchForLastPrefixKey(lastKeyPrefix)
		if err != nil {
			return 0, err
		}
		lastKeyPrefix = append(lastKeyPrefix, nextBiggestPrefixValue)
		// we have all 8 bytes of uint64 encoded block
		if len(lastKeyPrefix) == (stateHashPrefixSize + 8) {
			// full key is already found
			stateHashValue := lastKeyPrefix[stateHashPrefixSize:]

			if len(stateHashValue) != 8 {
				return 0, fmt.Errorf("undefined behaviour in value search; retrieved block bytes can't be converted")
			}
			var res uint64
			res, err = StateHashKeyToUint64(stateHashValue)
			if err != nil {
				return 0, err
			}
			return res, nil
		}
	}
}

// getLongestEncodedValue returns longest index of biggest block number to be search for in its search
func (db *SearchableDB) getLongestEncodedKeyZeroPrefixLength(dbPrefix string) (byte, error) {
	var i byte
	for i = 0; i < 8; i++ {
		startingIndex := make([]byte, 8)
		startingIndex[i] = 1
		if db.hasKeyValuesFor([]byte(dbPrefix), startingIndex) {
			return i, nil
		}
	}

	return 0, fmt.Errorf("unable to find prefix of state hash with biggest block")
}

func (db *SearchableDB) hasKeyValuesFor(prefix []byte, start []byte) bool {
	iter := db.NewIterator(prefix, start)
	defer iter.Release()
	return iter.Next()
}

func (db *SearchableDB) binarySearchForLastPrefixKey(lastKeyPrefix []byte) (byte, error) {
	var minimum uint16 = 0
	var maximum uint16 = 255

	startIndex := make([]byte, 1)

	for maximum-minimum > 1 {
		searchHalf := (maximum + minimum) / 2
		startIndex[0] = byte(searchHalf)
		if db.hasKeyValuesFor(lastKeyPrefix, startIndex) {
			minimum = searchHalf
		} else {
			maximum = searchHalf
		}
	}

	// shouldn't occure
	if maximum-minimum == 0 {
		return 0, fmt.Errorf("undefined behaviour in value search; maximum - minimum == 0")
	}

	startIndex[0] = byte(minimum)
	if db.hasKeyValuesFor(lastKeyPrefix, startIndex) {
		startIndex[0] = byte(maximum)
		if db.hasKeyValuesFor(lastKeyPrefix, startIndex) {
			return byte(maximum), nil
		} else {
			return byte(minimum), nil
		}
	} else {
		return 0, fmt.Errorf("no value found in search")
	}
}
