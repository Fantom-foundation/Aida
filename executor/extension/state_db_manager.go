package extension

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type StateDbManager struct {
	NilExtension
	config      *utils.Config
	StateDbPath string
	log         logger.Logger
}

// MakeStateDbManager creates a executor.Extension that commits state of StateDb if keep-db is enabled
func MakeStateDbManager(config *utils.Config) *StateDbManager {
	return &StateDbManager{
		config: config,
		log:    logger.NewLogger(config.LogLevel, "Db manager"),
	}
}

func (m *StateDbManager) PreRun(state executor.State, ctx *executor.Context) error {
	var err error
	ctx.State, m.StateDbPath, err = utils.PrepareStateDB(m.config)
	return err
}

func (m *StateDbManager) PostRun(state executor.State, ctx *executor.Context, _ error) error {
	if !m.config.KeepDb {
		ctx.State.Close()
		return os.RemoveAll(m.config.StateDbSrc)
	}

	rootHash := ctx.State.GetHash()
	if err := utils.WriteStateDbInfo(m.StateDbPath, m.config, uint64(state.Block), rootHash); err != nil {
		return fmt.Errorf("failed to create state-db info file; %v", err)
	}

	// stateDb needs to be closed between committing and renaming
	if err := ctx.State.Close(); err != nil {
		return fmt.Errorf("failed to close state-db; %v", err)
	}

	newName := utils.RenameTempStateDBDirectory(m.config, m.StateDbPath, uint64(state.Block))
	m.log.Noticef("State-db directory: %v", newName)
	return nil
}
