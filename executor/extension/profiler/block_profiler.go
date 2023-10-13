package profiler

import (
	"fmt"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/profile/blockprofile"
	"github.com/Fantom-foundation/Aida/utils"
)

func MakeBlockProfiler(cfg *utils.Config) executor.Extension {
	if !cfg.ProfileBlocks {
		return extension.NilExtension{}
	}
	return &BlockProfiler{
		cfg: cfg,
		log: logger.NewLogger(cfg.LogLevel, "Block-Profile"),
	}
}

type BlockProfiler struct {
	extension.NilExtension
	log        logger.Logger
	cfg        *utils.Config
	db         *blockprofile.ProfileDB
	ctx        *blockprofile.Context
	blockTimer time.Time
	txTimer    time.Time
}

// PreRun prepares the ProfileDB
func (b *BlockProfiler) PreRun(executor.State, *executor.Context) error {
	var err error
	b.db, err = blockprofile.NewProfileDB(b.cfg.ProfileDB)
	if err != nil {
		return fmt.Errorf("cannot create profile-db; %v", err)
	}

	b.log.Notice("Deleting old data from ProfileDB")
	_, err = b.db.DeleteByBlockRange(b.cfg.First, b.cfg.Last)
	if err != nil {
		return fmt.Errorf("cannot delete old data from profile-db; %v", err)
	}

	return nil
}

// PreTransaction resets the transaction timer.
func (b *BlockProfiler) PreTransaction(executor.State, *executor.Context) error {
	b.txTimer = time.Now()
	return nil
}

// PostTransaction records tx into profile context.
func (b *BlockProfiler) PostTransaction(state executor.State, _ *executor.Context) error {
	err := b.ctx.RecordTransaction(state, time.Since(b.txTimer))
	if err != nil {
		return fmt.Errorf("cannot record transaction; %v", err)
	}
	return nil
}

// PreBlock resets the block times and profile context.
func (b *BlockProfiler) PreBlock(executor.State, *executor.Context) error {
	b.ctx = blockprofile.NewContext()
	b.blockTimer = time.Now()
	return nil
}

// PostBlock extracts data from profile context and writes them to ProfileDB.
func (b *BlockProfiler) PostBlock(state executor.State, _ *executor.Context) error {
	data, err := b.ctx.GetProfileData(uint64(state.Block), time.Since(b.blockTimer))
	if err != nil {
		return fmt.Errorf("cannot get profile data from context; %v", err)
	}

	err = b.db.Add(*data)
	if err != nil {
		return fmt.Errorf("cannot add data to profile-db; %v", err)
	}

	return nil
}

// PostRun closes ProfileDB
func (b *BlockProfiler) PostRun(executor.State, *executor.Context, error) error {
	err := b.db.Close()
	if err != nil {
		return fmt.Errorf("cannot close profile-db; %v", err)
	}

	return nil
}
