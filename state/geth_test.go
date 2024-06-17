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
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const N = 1000

func fillDb(t *testing.T, directory string) (common.Hash, error) {
	db, err := MakeGethStateDB(directory, "", common.Hash{}, false, nil)
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	if err := db.BeginBlock(0); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}
	if err := db.BeginTransaction(0); err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}
	for i := 0; i < N; i++ {
		address := common.Address{byte(i), byte(i >> 8)}
		db.CreateAccount(address)
		db.SetNonce(address, 12)
		key := common.Hash{byte(i >> 8), byte(i)}
		value := common.Hash{byte(15)}
		db.SetState(address, key, value)
	}
	if err := db.EndTransaction(); err != nil {
		t.Fatalf("EndTransaction failed: %v", err)
	}
	if err := db.EndBlock(); err != nil {
		t.Fatalf("EndBlock failed: %v", err)
	}
	hash, err := db.GetHash()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if err = db.Close(); err != nil {
		t.Fatalf("Failed to close DB: %v", err)
	}
	return hash, nil
}

func TestGethDbFilling(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_db_*")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", dir)
	}
	if _, err := fillDb(t, dir); err != nil {
		t.Errorf("Unable to fill DB: %v", err)
	}
}

func TestGethDbReloadData(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_db_*")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", dir)
	}
	hash, err := fillDb(t, dir)
	if err != nil {
		t.Errorf("Unable to fill DB: %v", err)
	}

	// Re-open the data base.
	db, err := MakeGethStateDB(dir, "", hash, false, nil)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	for i := 0; i < N; i++ {
		address := common.Address{byte(i), byte(i >> 8)}
		if got := db.GetNonce(address); got != 12 {
			t.Fatalf("Nonce of %v is not 12: %v", address, got)
		}
		key := common.Hash{byte(i >> 8), byte(i)}
		value := common.Hash{byte(15)}
		if got := db.GetState(address, key); got != value {
			t.Fatalf("Value of %v/%v is not %v: %v", address, key, value, got)
		}
	}
	if err = db.Close(); err != nil {
		t.Fatalf("Failed to close DB: %v", err)
	}
}
