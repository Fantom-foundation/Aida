package extension

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type dbManager struct {
	NilExtension
	config *utils.Config
	log    logger.Logger
}

// MakeDbManager creates a executor.Extension that commits state of StateDb if keep-db is enabled
func MakeDbManager(config *utils.Config) executor.Extension {
	if !config.KeepDb {
		return NilExtension{}
	}

	return &dbManager{
		config: config,
		log:    logger.NewLogger(config.LogLevel, "Db manager"),
	}
}

func (m *dbManager) PostRun(state executor.State, _ error) error {
	rootHash, _ := state.State.Commit(true)
	if err := utils.WriteStateDbInfo(m.config.StateDbSrc, m.config, uint64(state.Block), rootHash); err != nil {
		return fmt.Errorf("failed to create state-db info file; %v", err)
	}

	newName := utils.RenameTempStateDBDirectory(m.config, m.config.StateDbSrc, uint64(state.Block))
	m.log.Noticef("State-db directory: %v", newName)

	return nil
}
