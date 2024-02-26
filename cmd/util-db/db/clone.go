package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// CloneCommand clones aida-db as standalone or patch database
var CloneCommand = cli.Command{
	Name:  "clone",
	Usage: `Used for creation of standalone subset of aida-db or patch`,
	Subcommands: []*cli.Command{
		&CloneDb,
		&ClonePatch,
		&CloneCustom,
	},
}

// ClonePatch enables creation of aida-db read or subset
var ClonePatch = cli.Command{
	Action:    clonePatch,
	Name:      "patch",
	Usage:     "patch is used to create aida-db patch",
	ArgsUsage: "<blockNumFirst> <blockNumLast> <EpochNumFirst> <EpochNumLast>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.TargetDbFlag,
		&utils.CompactDbFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates patch of aida-db for desired block range
`,
}

// CloneDb enables creation of aida-db read or subset
var CloneDb = cli.Command{
	Action:    createDbClone,
	Name:      "db",
	Usage:     "clone db creates aida-db subset",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.TargetDbFlag,
		&utils.CompactDbFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates clone db is used to create subset of aida-db to have more compact database, but still fully usable for desired block range.
`,
}

// CloneDb enables creation of aida-db read or subset
var CloneCustom = cli.Command{
	Action:    createCustomClone,
	Name:      "custom",
	Usage:     "clone db creates aida-db custom subset",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.DbComponentFlag,
		&utils.TargetDbFlag,
		&utils.CompactDbFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates custom clone db is used to create subset of aida-db with only specific component.
`,
}

// clonePatch creates aida-db patch
func clonePatch(ctx *cli.Context) error {
	// TODO refactor
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	if ctx.Args().Len() != 4 {
		return fmt.Errorf("clone patch command requires exactly 4 arguments")
	}

	cfg.First, cfg.Last, err = utils.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1), cfg.ChainID)
	if err != nil {
		return err
	}

	var firstEpoch, lastEpoch uint64
	firstEpoch, lastEpoch, err = utils.SetBlockRange(ctx.Args().Get(2), ctx.Args().Get(3), cfg.ChainID)
	if err != nil {
		return err
	}

	aidaDb, targetDb, err := utildb.OpenCloningDbs(cfg.AidaDb, cfg.TargetDb)
	if err != nil {
		return err
	}

	err = utildb.CreatePatchClone(cfg, aidaDb, targetDb, firstEpoch, lastEpoch, false)
	if err != nil {
		return err
	}

	utildb.MustCloseDB(aidaDb)
	utildb.MustCloseDB(targetDb)

	return utildb.PrintMetadata(cfg.TargetDb)
}

// createDbClone creates aida-db copy or subset
func createDbClone(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	aidaDb, targetDb, err := utildb.OpenCloningDbs(cfg.AidaDb, cfg.TargetDb)
	if err != nil {
		return err
	}

	err = utildb.Clone(cfg, aidaDb, targetDb, utils.CloneType, false)
	if err != nil {
		return err
	}

	utildb.MustCloseDB(aidaDb)
	utildb.MustCloseDB(targetDb)

	return utildb.PrintMetadata(cfg.TargetDb)
}

// createDbClone creates aida-db copy or subset
func createCustomClone(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	aidaDb, targetDb, err := utildb.OpenCloningDbs(cfg.AidaDb, cfg.TargetDb)
	if err != nil {
		return err
	}

	err = utildb.Clone(cfg, aidaDb, targetDb, utils.CustomType, false)
	if err != nil {
		return err
	}

	utildb.MustCloseDB(aidaDb)
	utildb.MustCloseDB(targetDb)

	return nil
}
