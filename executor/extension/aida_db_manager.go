package extension

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// MakeAidaDbManager opens AidaDb if path is given and adds it to the context.
func MakeAidaDbManager(cfg *utils.Config) executor.Extension {
	if cfg.AidaDb != "" {
		return &AidaDbManager{path: cfg.AidaDb}
	}
	return nil
}

type AidaDbManager struct {
	NilExtension
	path string
}

func (e *AidaDbManager) PreRun(_ executor.State, context *executor.Context) error {
	db, err := rawdb.NewLevelDBDatabase(e.path, 1024, 100, "", true)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}
	context.AidaDb = db

	return nil
}

func (e *AidaDbManager) PostRun(_ executor.State, context *executor.Context, _ error) error {
	if err := context.AidaDb.Close(); err != nil {
		return fmt.Errorf("cannot close AidaDb; %v", err)
	}

	return nil
}
