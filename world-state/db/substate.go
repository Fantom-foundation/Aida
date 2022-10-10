// Package db implements database interfaces for the world state manager.
package db

import (
	"github.com/ethereum/go-ethereum/core/rawdb"
	"log"
)

// SubstateDB represents the state snapshot database handle.
type SubstateDB struct {
	Backend BackendDatabase
}

// OpenSubstateDB opens substate database at the given path.
func OpenSubstateDB(path string) (*SubstateDB, error) {
	backend, err := rawdb.NewLevelDBDatabase(path, 1024, 100, "substatedir", false)
	if err != nil {
		return nil, err
	}

	return &SubstateDB{Backend: backend}, nil
}

// MustCloseSubstateDB closes the substate database without raising an error.
func MustCloseSubstateDB(db *SubstateDB) {
	if db != nil {
		err := db.Backend.Close()
		if err != nil {
			log.Printf("could not close state snapshot; %s\n", err.Error())
		}
	}
}
