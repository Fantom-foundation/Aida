package db

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

var MetadataCommand = cli.Command{
	Name:  "metadata",
	Usage: "Does action with AidaDb metadata",
	Subcommands: []*cli.Command{
		&cmdPrintMetadata,
		&cmdGenerateMetadata,
	},
}

var cmdPrintMetadata = cli.Command{
	Action: printMetadata,
	Name:   "print",
	Usage:  "Prints metadata",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

func printMetadata(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	return utildb.PrintMetadata(cfg.AidaDb)
}

var cmdGenerateMetadata = cli.Command{
	Action: generateMetadata,
	Name:   "generate",
	Usage:  "Generates new metadata for given chain-id",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
	},
}

func generateMetadata(ctx *cli.Context) error {
	cfg, argErr := utils.NewConfig(ctx, utils.NoArgs)
	if argErr != nil {
		return argErr
	}

	substate.SetSubstateDb(cfg.AidaDb)
	substate.OpenSubstateDB()
	fb, lb, ok := utils.FindBlockRangeInSubstate()
	if !ok {
		return errors.New("cannot find block range in substate")
	}
	substate.CloseSubstateDB()

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")
	md.FirstBlock = fb
	md.LastBlock = lb
	if err = md.SetFreshMetadata(cfg.ChainID); err != nil {
		return err
	}

	if err = aidaDb.Close(); err != nil {
		return fmt.Errorf("cannot close aida-db; %v", err)
	}

	return utildb.PrintMetadata(cfg.AidaDb)

}
