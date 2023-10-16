package extension

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// MakeAidaDbManager opens AidaDb if path is given and adds it to the context.
func MakeAidaDbManager[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.AidaDb == "" {
		return NilExtension[T]{}
	}
	return &AidaDbManager[T]{path: cfg.AidaDb}
}

type AidaDbManager[T any] struct {
	NilExtension[T]
	path string
}

func (e *AidaDbManager[T]) PreRun(_ executor.State[T], context *executor.Context) error {
	db, err := rawdb.NewLevelDBDatabase(e.path, 1024, 100, "", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}
	context.AidaDb = db

	return nil
}

func (e *AidaDbManager[T]) PostRun(_ executor.State[T], context *executor.Context, _ error) error {
	if err := context.AidaDb.Close(); err != nil {
		return fmt.Errorf("cannot close AidaDb; %v", err)
	}

	context.AidaDb = nil

	return nil
}
