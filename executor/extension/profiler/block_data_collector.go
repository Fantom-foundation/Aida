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

func MakeBlockDataCollector(cfg *utils.Config) executor.Extension {
	if !cfg.ProfileBlocks {
		return extension.NilExtension{}
	}
	return &BlockDataCollector{
		cfg: cfg,
		log: logger.NewLogger(cfg.LogLevel, "Block-Profile"),
	}
}

type BlockDataCollector struct {
	extension.NilExtension
	log        logger.Logger
	cfg        *utils.Config
	profileDb  *blockprofile.ProfileDB
	ctx        *blockprofile.Context
	blockTimer time.Time
	txTimer    time.Time
}

// PreRun prepares the ProfileDB
func (b *BlockDataCollector) PreRun(executor.State, *executor.Context) error {
	var err error
	b.profileDb, err = blockprofile.NewProfileDB(b.cfg.ProfileDB)
	if err != nil {
		return fmt.Errorf("cannot create profile-db; %v", err)
	}

	b.log.Notice("Deleting old data from ProfileDB")
	_, err = b.profileDb.DeleteByBlockRange(b.cfg.First, b.cfg.Last)
	if err != nil {
		return fmt.Errorf("cannot delete old data from profile-db; %v", err)
	}

	return nil
}

// PreTransaction resets the transaction timer.
func (b *BlockDataCollector) PreTransaction(executor.State, *executor.Context) error {
	b.txTimer = time.Now()
	return nil
}

// PostTransaction records tx into profile context.
func (b *BlockDataCollector) PostTransaction(state executor.State, _ *executor.Context) error {
	err := b.ctx.RecordTransaction(state, time.Since(b.txTimer))
	if err != nil {
		return fmt.Errorf("cannot record transaction; %v", err)
	}
	return nil
}

// PreBlock resets the block times and profile context.
func (b *BlockDataCollector) PreBlock(executor.State, *executor.Context) error {
	b.ctx = blockprofile.NewContext()
	b.blockTimer = time.Now()
	return nil
}

// PostBlock extracts data from profile context and writes them to ProfileDB.
func (b *BlockDataCollector) PostBlock(state executor.State, _ *executor.Context) error {
	data, err := b.ctx.GetProfileData(uint64(state.Block), time.Since(b.blockTimer))
	if err != nil {
		return fmt.Errorf("cannot get profile data from context; %v", err)
	}

	err = b.profileDb.Add(*data)
	if err != nil {
		return fmt.Errorf("cannot add data to profile-db; %v", err)
	}

	return nil
}

// PostRun closes ProfileDB
func (b *BlockDataCollector) PostRun(executor.State, *executor.Context, error) error {
	defer func() {
		if r := recover(); r != nil {
			b.log.Errorf("recovered panic in block-profiler; %v", r)
		}
	}()

	err := b.profileDb.Close()
	if err != nil {
		return fmt.Errorf("cannot close profile-db; %v", err)
	}

	return nil
}