package substate

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"io"
)

type SubstateDB struct {
	Backend BackendDatabase
}

type BackendDatabase interface {
	ethdb.KeyValueReader
	ethdb.KeyValueWriter
	ethdb.Batcher
	ethdb.Iteratee
	ethdb.Stater
	ethdb.Compacter
	io.Closer
}

func OpenSubstateDB(substateDatadir string) *SubstateDB {
	backend, err := rawdb.NewLevelDBDatabase(substateDatadir, 1024, 100, "substatedir", false)
	if err != nil {
		panic(fmt.Errorf("error opening substate leveldb %s: %v", substateDatadir, err))
	}

	return &SubstateDB{Backend: backend}
}
