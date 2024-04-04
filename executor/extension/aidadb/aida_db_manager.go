package aidadb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
)

// MakeAidaDbManager opens AidaDb if path is given and adds it to the context.
func MakeAidaDbManager[T any](cfg *utils.Config) executor.Extension[T] {
	if cfg.AidaDb == "" {
		return extension.NilExtension[T]{}
	}
	return &AidaDbManager[T]{path: cfg.AidaDb}
}

type AidaDbManager[T any] struct {
	extension.NilExtension[T]
	path string
}

func (e *AidaDbManager[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	database, err := db.NewDefaultBaseDB(e.path)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}
	ctx.AidaDb = database

	return nil
}

func (e *AidaDbManager[T]) PostRun(_ executor.State[T], ctx *executor.Context, _ error) error {
	if err := ctx.AidaDb.Close(); err != nil {
		return fmt.Errorf("cannot close AidaDb; %v", err)
	}

	ctx.AidaDb = nil

	return nil
}
