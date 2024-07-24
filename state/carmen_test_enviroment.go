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

package state

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Fantom-foundation/Carmen/go/carmen"
	_ "github.com/Fantom-foundation/Carmen/go/carmen/experimental"
	_ "github.com/Fantom-foundation/Carmen/go/state/cppstate"
	_ "github.com/Fantom-foundation/Carmen/go/state/gostate"
)

type CarmenStateTestCase struct {
	Variant string
	Schema  int
	Archive string
}

func NewCarmenStateTestCase(variant string, schema int, archive string) CarmenStateTestCase {
	return CarmenStateTestCase{Variant: variant, Schema: schema, Archive: archive}
}

func (c CarmenStateTestCase) String() string {
	return fmt.Sprintf("DB Variant: %s, Schema: %d, Archive type: %v", c.Variant, c.Schema, c.Archive)
}

// A combination of all carmen db configurations for testing interface
func GetAllCarmenConfigurations() []CarmenStateTestCase {
	var res []CarmenStateTestCase

	for _, cfg := range carmen.GetAllConfigurations() {
		res = append(res, NewCarmenStateTestCase(string(cfg.Variant), int(cfg.Schema), string(cfg.Archive)))
	}
	return res
}

// GetCurrentCarmenTestCases returns currently used carmen version.
func GetCurrentCarmenTestCases() []CarmenStateTestCase {
	var res []CarmenStateTestCase

	for _, cfg := range carmen.GetAllConfigurations() {
		if cfg.Variant != "go-file" {
			continue
		}
		if cfg.Schema != 5 {
			continue
		}
		if cfg.Archive != "ldb" && cfg.Archive != "leveldb" && cfg.Archive != "none" {
			continue
		}
		res = append(res, NewCarmenStateTestCase(string(cfg.Variant), int(cfg.Schema), string(cfg.Archive)))
	}
	return res
}

// A minimal combination of carmen db configuration for testing interface
func GetCarmenStateTestCases() []CarmenStateTestCase {
	return GetCurrentCarmenTestCases()
}

// MakeRandomByteSlice creates byte slice of given length with randomized values
func MakeRandomByteSlice(t *testing.T, bufferLength int) []byte {
	// make byte slice
	buffer := make([]byte, bufferLength)

	// fill the slice with random data
	_, err := rand.Read(buffer)
	if err != nil {
		t.Fatalf("failed test data; can not generate random byte slice; %s", err.Error())
	}

	return buffer
}

// GetRandom produces a random uint in range [lower, upper)
func GetRandom(t *testing.T, lower int, upper int) uint64 {
	if upper < lower {
		t.Fatalf("invalid range. The upper bound, %v, is smaller than the lower bound, %v.", upper, lower)
	}
	if lower < 0 || upper < 0 {
		t.Fatalf("expected positive integer. (%v, %v)", lower, upper)
	}
	// seed the PRNG
	rand.Seed(time.Now().UnixNano())

	// get randomized balance
	return uint64(lower + rand.Intn(upper-lower))
}

func MakeCarmenDbTestContext(dir string, variant string, schema int, archive string) (StateDB, error) {
	db, err := MakeCarmenStateDB(dir, variant, schema, archive)
	if err != nil {
		return nil, err
	}

	err = BeginCarmenDbTestContext(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func CloseCarmenDbTestContext(db StateDB) error {
	err := db.EndTransaction()
	if err != nil {
		return err
	}
	err = db.EndBlock()
	if err != nil {
		return err
	}
	return db.Close()
}

func BeginCarmenDbTestContext(db StateDB) error {
	err := db.BeginBlock(uint64(6))
	if err != nil {
		return fmt.Errorf("cannot begin block; %w", err)
	}

	err = db.BeginTransaction(uint32(0))
	if err != nil {
		return fmt.Errorf("cannot begin transaction; %w", err)
	}

	return nil
}
