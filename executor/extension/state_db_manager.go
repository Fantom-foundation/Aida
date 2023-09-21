package extension

import (
	"errors"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type stateDbManager struct {
	NilExtension
	config      *utils.Config
	stateDbPath string
	log         logger.Logger
}

// MakeStateDbManager creates a executor.Extension that commits state of StateDb if keep-db is enabled
func MakeStateDbManager(config *utils.Config) *stateDbManager {
	return &stateDbManager{
		config: config,
		log:    logger.NewLogger(config.LogLevel, "Db manager"),
	}
}

func (m *stateDbManager) PreRun(state executor.State, ctx *executor.Context) error {
	var err error
	ctx.State, m.stateDbPath, err = utils.PrepareStateDB(m.config)
	if !m.config.KeepDb {
		m.log.Warningf("--keep-db is not used. Directory %v with DB will be removed at the end of this run.", m.stateDbPath)
	}
	return err
}

func (m *stateDbManager) PostRun(state executor.State, ctx *executor.Context, _ error) error {
	//  if state was not correctly initialized remove the stateDbPath and abort
	if ctx.State == nil {
		var err = fmt.Errorf("state-db is nil")

		if !m.config.SrcDbReadonly {
			err = errors.Join(err, os.RemoveAll(m.stateDbPath))
		}
		return err
	}

	// if db isn't kept, then close and delete temporary state-db
	if !m.config.KeepDb {
		if err := ctx.State.Close(); err != nil {
			return fmt.Errorf("failed to close state-db; %v", err)
		}

		if !m.config.SrcDbReadonly {
			return os.RemoveAll(m.stateDbPath)
		}
		return nil
	}

	if m.config.SrcDbReadonly {
		m.log.Noticef("State-db directory was readonly %v", m.stateDbPath)
		return nil
	}

	rootHash := ctx.State.GetHash()
	if err := utils.WriteStateDbInfo(m.stateDbPath, m.config, uint64(state.Block), rootHash); err != nil {
		return fmt.Errorf("failed to create state-db info file; %v", err)
	}

	// stateDb needs to be closed between committing and renaming
	if err := ctx.State.Close(); err != nil {
		return fmt.Errorf("failed to close state-db; %v", err)
	}

	newName := utils.RenameTempStateDBDirectory(m.config, m.stateDbPath, uint64(state.Block))
	m.log.Noticef("State-db directory: %v", newName)
	return nil
}
