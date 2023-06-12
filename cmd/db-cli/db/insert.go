package db

import (
	"fmt"
	"strconv"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

// InsertKeyValueCommand is a generic command for inserting any key/value pair into AidaDb
var InsertKeyValueCommand = cli.Command{
	Action: insertKeyValue,
	Name:   "insertKeyValue",
	Usage:  "insert key/value pair into AidaDb",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
	Description: `
Inserts key/value pair into AidaDb according to arguments:
<key> <value>
`,
}

// insertKeyValue given key/value pair into AidaDb
func insertKeyValue(ctx *cli.Context) error {
	var (
		err error
		u   uint64
	)

	aidaDbPath := ctx.String(utils.AidaDbFlag.Name)

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("this command requires two arguments - <key> <value>")
	}

	key := ctx.Args().Get(0)
	val := ctx.Args().Get(1)

	// open aidaDb
	aidaDb, err := rawdb.NewLevelDBDatabase(aidaDbPath, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb. Error: %v", err)
	}

	defer MustCloseDB(aidaDb)

	m := newAidaMetadata(aidaDb, noType, "INFO")

	switch substate.MetadataPrefix + key {
	case FirstBlockPrefix:
		u, err = strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", val, err)
		}
		m.setFirstBlock(u)
	case LastBlockPrefix:
		u, err = strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", val, err)
		}
		m.setLastBlock(u)
	case FirstEpochPrefix:
		u, err = strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", val, err)
		}
		m.setFirstEpoch(u)
	case LastEpochPrefix:
		u, err = strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", val, err)
		}
		m.setLastEpoch(u)
	case TypePrefix:
		m.setDbType(aidaDbType([]byte(val)[0]))
	case ChainIDPrefix:
		u, err = strconv.ParseUint(val, 10, 16)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", val, err)
		}
		m.setChainID(int(u))
	default:
		return fmt.Errorf("incorrect key: %v", key)
	}

	return nil
}
