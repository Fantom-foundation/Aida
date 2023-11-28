package db

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utildb"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

// signatureCmd creates signatures of substates, updatesets, deletion and state-hashes.
func signatureCmd(ctx *cli.Context) error {
	// process arguments and flags
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("signature command requires exactly 1 arguments")
	}
	cfg, err := utils.NewConfig(ctx, utils.LastBlockArg)
	if err != nil {
		return err
	}

	// if source db doesn't exist raise error
	_, err = os.Stat(cfg.AidaDb)
	if os.IsNotExist(err) {
		return fmt.Errorf("specified aida-db %v is empty\n", cfg.AidaDb)
	}
	// open db
	var aidaDb ethdb.Database
	aidaDb, err = rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("aidaDb %v; %v", cfg.AidaDb, err)
	}

	log := logger.NewLogger(cfg.LogLevel, "signature")
	log.Info("Inspecting database...")
	err = utildb.DbSignature(cfg, aidaDb, log)
	if err != nil {
		return err
	}
	log.Info("Finished")

	utildb.MustCloseDB(aidaDb)
	return nil
}
