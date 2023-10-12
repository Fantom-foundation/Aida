package extension

import (
	"fmt"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeStateHashValidator(config *utils.Config) executor.Extension {
	if !config.ValidateStateHashes {
		return NilExtension{}
	}

	log := logger.NewLogger("INFO", "state-hash-validator")
	return makeStateHashValidator(config, log)
}

func makeStateHashValidator(config *utils.Config, log logger.Logger) *stateHashValidator {
	return &stateHashValidator{config: config, log: log}
}

type stateHashValidator struct {
	NilExtension
	config                  *utils.Config
	log                     logger.Logger
	nextArchiveBlockToCheck int
	lastProcessedBlock      int
	hashProvider            utils.StateHashProvider
}

func (e *stateHashValidator) PreRun(_ executor.State, ctx *executor.Context) error {
	e.hashProvider = utils.MakeStateHashProvider(ctx.AidaDb, e.log)
	err := e.hashProvider.PreLoadStateHashes(int(e.config.First), int(e.config.Last))
	if err != nil {
		return err
	}
	return nil
}

func (e *stateHashValidator) PostBlock(state executor.State, context *executor.Context) error {
	if context.State == nil {
		return nil
	}

	want := e.hashProvider.GetStateHash(state.Block)

	got := context.State.GetHash()
	if want != got {
		return fmt.Errorf("unexpected hash for Live block %d\nwanted %v\n   got %v", state.Block, want, got)
	}

	// Check the ArchiveDB
	if e.config.ArchiveMode {
		e.lastProcessedBlock = state.Block
		if err := e.checkArchiveHashes(context.State); err != nil {
			return err
		}
	} else {
		// delete the record only if archive is enabled, since archive can be delayed,
		// so delete for this case is done inside e.checkArchiveHashes
		e.hashProvider.DeletePreLoadedStateHash(state.Block)

	}

	return nil
}

func (e *stateHashValidator) PostRun(_ executor.State, context *executor.Context, err error) error {
	// Skip processing if run is aborted due to an error.
	if err != nil {
		return nil
	}
	// Complete processing remaining archive blocks.
	if e.config.ArchiveMode {
		for e.nextArchiveBlockToCheck < e.lastProcessedBlock {
			if err = e.checkArchiveHashes(context.State); err != nil {
				return err
			}
			if e.nextArchiveBlockToCheck < e.lastProcessedBlock {
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
	return nil
}

func (e *stateHashValidator) checkArchiveHashes(state state.StateDB) error {
	// Note: the archive may be lagging behind the life DB, so block hashes need
	// to be checked as they become available.
	height, empty, err := state.GetArchiveBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get archive block height: %v", err)
	}

	cur := uint64(e.nextArchiveBlockToCheck)
	for !empty && cur <= height {

		want := e.hashProvider.GetStateHash(int(cur))

		archive, err := state.GetArchiveState(cur)
		if err != nil {
			return err
		}

		got := archive.GetHash()
		archive.Release()
		if want != got {
			return fmt.Errorf("unexpected hash for archive block %d\nwanted %v\n   got %v", cur, want, got)
		}

		// if archive is enabled delete the record only after it's been used for archive check
		e.hashProvider.DeletePreLoadedStateHash(int(cur))

		cur++
	}
	e.nextArchiveBlockToCheck = int(cur)
	return nil
}
