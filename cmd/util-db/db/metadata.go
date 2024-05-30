// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package db

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

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
		&InsertMetadataCommand,
		&RemoveMetadataCommand,
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

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open aida-db; %v", err)
	}
	substate.SetSubstateDbBackend(aidaDb)
	fb, lb, ok := utils.FindBlockRangeInSubstate()
	if !ok {
		return errors.New("cannot find block range in substate")
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

// InsertMetadataCommand is a generic command for inserting any metadata key/value pair into AidaDb
var InsertMetadataCommand = cli.Command{
	Action: insertMetadata,
	Name:   "insert",
	Usage:  "inserts key/value metadata pair into AidaDb",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
	Description: `
Inserts key/value pair into AidaDb according to arguments:
<key> <value>
If given key is not metadata-key, operation fails.
`,
}

// insertMetadata key/value pair into AidaDb
func insertMetadata(ctx *cli.Context) error {
	var (
		err error
		val uint64
	)

	aidaDbPath := ctx.String(utils.AidaDbFlag.Name)

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("this command requires two arguments - <keyArg> <value>")
	}

	keyArg := ctx.Args().Get(0)
	valArg := ctx.Args().Get(1)

	// open db
	aidaDb, err := rawdb.NewLevelDBDatabase(aidaDbPath, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb. Error: %v", err)
	}

	defer utildb.MustCloseDB(aidaDb)

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")

	switch substate.MetadataPrefix + keyArg {
	case utils.FirstBlockPrefix:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.SetFirstBlock(val); err != nil {
			return err
		}
	case utils.LastBlockPrefix:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.SetLastBlock(val); err != nil {
			return err
		}
	case utils.FirstEpochPrefix:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.SetFirstEpoch(val); err != nil {
			return err
		}
	case utils.LastEpochPrefix:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.SetLastEpoch(val); err != nil {
			return err
		}
	case utils.TypePrefix:
		num64, err := strconv.ParseUint(valArg, 10, 8)
		if err != nil {
			return err
		} 
		if err = md.SetDbType(utils.AidaDbType(uint8(num64))); err != nil {
			return err
		}
	case utils.ChainIDPrefix:
		val, err = strconv.ParseUint(valArg, 10, 16)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.SetChainID(utils.ChainID(val)); err != nil {
			return err
		}
	case utils.TimestampPrefix:
		if err = md.SetTimestamp(); err != nil {
			return err
		}
	case utils.DbHashPrefix:
		hash, err := hex.DecodeString(valArg)
		if err != nil {
			return fmt.Errorf("cannot decode db-hash string into []byte; %v", err)
		}
		if err = md.SetDbHash(hash); err != nil {
			return err
		}
	case substate.UpdatesetIntervalKey:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.SetUpdatesetInterval(val); err != nil {
			return err
		}
	case substate.UpdatesetSizeKey:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.SetUpdatesetSize(val); err != nil {
			return err
		}
	default:
		return fmt.Errorf("incorrect keyArg: %v", keyArg)
	}

	return nil
}

// RemoveMetadataCommand is a command used for creating testing environment without metadata
var RemoveMetadataCommand = cli.Command{
	Action: removeMetadata,
	Name:   "remove",
	Usage:  "remove metadata from aidaDb",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
	Description: `
Removes block and epoch range and ChainID from metadata for given AidaDb.
`,
}

// removeMetadata command is used for testing scenario where AidaDb does not have metadata and a patch
// is applied onto it
func removeMetadata(ctx *cli.Context) error {
	aidaDbPath := ctx.String(utils.AidaDbFlag.Name)

	// open db
	aidaDb, err := rawdb.NewLevelDBDatabase(aidaDbPath, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb. Error: %v", err)
	}

	md := utils.NewAidaDbMetadata(aidaDb, "DEBUG")
	md.DeleteMetadata()

	return nil
}
