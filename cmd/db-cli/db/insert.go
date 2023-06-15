package db

import (
	"fmt"
	"strconv"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

// InsertMetadataCommand is a generic command for inserting any metadata key/value pair into AidaDb
var InsertMetadataCommand = cli.Command{
	Action: insertMetadata,
	Name:   "insert-metadata",
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

// RemoveMetadataCommand is a command used for creating testing environment without metadata
var RemoveMetadataCommand = cli.Command{
	Action: removeMetadata,
	Name:   "remove-metadata",
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

	if err = aidaDb.Delete([]byte(ChainIDPrefix)); err != nil {
		return err
	}
	if err = aidaDb.Delete([]byte(FirstBlockPrefix)); err != nil {
		return err
	}
	if err = aidaDb.Delete([]byte(LastBlockPrefix)); err != nil {
		return err
	}
	if err = aidaDb.Delete([]byte(FirstEpochPrefix)); err != nil {
		return err
	}
	if err = aidaDb.Delete([]byte(LastEpochPrefix)); err != nil {
		return err
	}
	return nil
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

	defer MustCloseDB(aidaDb)

	md := newAidaMetadata(aidaDb, "INFO")

	switch substate.MetadataPrefix + keyArg {
	case FirstBlockPrefix:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.setFirstBlock(val); err != nil {
			return err
		}
	case LastBlockPrefix:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.setLastBlock(val); err != nil {
			return err
		}
	case FirstEpochPrefix:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.setFirstEpoch(val); err != nil {
			return err
		}
	case LastEpochPrefix:
		val, err = strconv.ParseUint(valArg, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.setLastEpoch(val); err != nil {
			return err
		}
	case TypePrefix:
		num, err := strconv.Atoi(valArg)
		if err != nil {
			return err
		}
		if err = md.setDbType(aidaDbType(num)); err != nil {
			return err
		}
	case ChainIDPrefix:
		val, err = strconv.ParseUint(valArg, 10, 16)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", valArg, err)
		}
		if err = md.setChainID(int(val)); err != nil {
			return err
		}
	case TimestampPrefix:
		if err = md.setTimestamp(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("incorrect keyArg: %v", keyArg)
	}

	return nil
}
