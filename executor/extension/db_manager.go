package extension

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

type dbManager struct {
	NilExtension
	config *utils.Config
}

// MakeDbManager creates a executor.Extension that commits state of StateDb if keep-db is enabled
func MakeDbManager(config *utils.Config) executor.Extension {
	if !config.KeepDb {
		return NilExtension{}
	}

	return &dbManager{
		config: config,
	}
}

func (p *dbManager) PostRun(state executor.State, _ error) error {
	rootHash, _ := state.State.Commit(true)
	if err := utils.WriteStateDbInfo(p.config.StateDbSrc, p.config, uint64(state.Block), rootHash); err != nil {
		return fmt.Errorf("failed to create state-db info file; %v", err)
	}

	_ = utils.RenameTempStateDBDirectory(p.config, p.config.StateDbSrc, uint64(state.Block))

	return nil
}
