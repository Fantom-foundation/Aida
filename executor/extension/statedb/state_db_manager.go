package statedb

import (
	"errors"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

// MakeStateDbManager creates a executor.Extension that commits state of StateDb if keep-db is enabled
func MakeStateDbManager[T any](cfg *utils.Config) executor.Extension[T] {
	return &stateDbManager[T]{
		cfg: cfg,
		log: logger.NewLogger(cfg.LogLevel, "Db manager"),
	}
}

type stateDbManager[T any] struct {
	extension.NilExtension[T]
	cfg *utils.Config
	log logger.Logger
}

func (m *stateDbManager[T]) PreRun(_ executor.State[T], ctx *executor.Context) error {
	var err error
	if ctx.State == nil {
		ctx.State, ctx.StateDbPath, err = utils.PrepareStateDB(m.cfg)
		if err != nil {
			return err
		}
	}

	if !m.cfg.ShadowDb {
		m.logDbMode("Db Implementation", m.cfg.DbImpl, m.cfg.DbVariant)
	} else {
		m.logDbMode("Prime Db Implementation", m.cfg.DbImpl, m.cfg.DbVariant)
		m.logDbMode("Shadow Db Implementation", m.cfg.ShadowImpl, m.cfg.ShadowVariant)
	}

	if m.cfg.StateDbSrc != "" {
		// if using pre-existing StateDb and running in read-only mode, we must report both source db and working tmp dir
		m.log.Infof("Source storage directory: %v", m.cfg.StateDbSrc)
		if m.cfg.SrcDbReadonly {
			m.log.Infof("Working storage directory: %v", m.cfg.DbTmp)
		}

	} else {
		// otherwise only working directory is reported
		m.log.Infof("Working storage directory: %v", m.cfg.DbTmp)
	}

	if m.cfg.ArchiveMode {
		var archiveVariant string
		if m.cfg.ArchiveVariant == "" {
			archiveVariant = "<implementation-default>"
		} else {
			archiveVariant = m.cfg.ArchiveVariant
		}

		m.log.Noticef("Archive mode enabled; Variant: %v", archiveVariant)

	} else {
		m.log.Infof("Archive mode disabled")
	}

	if !m.cfg.KeepDb {
		m.log.Warningf("--keep-db is not used. Directory %v with DB will be removed at the end of this run.", ctx.StateDbPath)
	}
	return nil
}

func (m *stateDbManager[T]) PostRun(state executor.State[T], ctx *executor.Context, _ error) error {
	//  if state was not correctly initialized remove the stateDbPath and abort
	if ctx.State == nil {
		var err = fmt.Errorf("state-db is nil")
		if !m.cfg.SrcDbReadonly {
			err = errors.Join(err, os.RemoveAll(ctx.StateDbPath))
		}
		return err
	}

	// if db isn't kept, then close and delete temporary state-db
	if !m.cfg.KeepDb {
		if err := ctx.State.Close(); err != nil {
			return fmt.Errorf("failed to close state-db; %v", err)
		}

		if !m.cfg.SrcDbReadonly {
			return os.RemoveAll(ctx.StateDbPath)
		}
		return nil
	}

	if m.cfg.SrcDbReadonly {
		m.log.Noticef("State-db directory was readonly %v", ctx.StateDbPath)
		return nil
	}

	// lastProcessedBlock contains number of last successfully processed block
	// - processing finished successfully to the end, but then state.Block is set to params.To
	// - error occurred therefore previous block is last successful
	lastProcessedBlock := uint64(state.Block)
	if lastProcessedBlock > 0 {
		lastProcessedBlock -= 1
	}

	rootHash := ctx.State.GetHash()
	if err := utils.WriteStateDbInfo(ctx.StateDbPath, m.cfg, lastProcessedBlock, rootHash); err != nil {
		return fmt.Errorf("failed to create state-db info file; %v", err)
	}

	// stateDb needs to be closed between committing and renaming
	if err := ctx.State.Close(); err != nil {
		return fmt.Errorf("failed to close state-db; %v", err)
	}

	newName := utils.RenameTempStateDbDirectory(m.cfg, ctx.StateDbPath, lastProcessedBlock)
	m.log.Noticef("State-db directory: %v", newName)
	return nil
}

func (m *stateDbManager[T]) logDbMode(prefix, impl, variant string) {
	if m.cfg.DbImpl == "carmen" {
		m.log.Noticef("%s: %v; Variant: %v, Carmen Schema: %d", prefix, impl, variant, m.cfg.CarmenSchema)
	} else {
		m.log.Noticef("%s: %v; Variant: %v", prefix, impl, variant)
	}
}
