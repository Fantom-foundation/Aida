package db

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/urfave/cli/v2"
)

// InsertCommand is a generic command for inserting any key/value pair into AidaDb
var InsertCommand = cli.Command{
	Action: insert,
	Name:   "insert",
	Usage:  "insert key/value pair into AidaDb",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
	Description: `
Inserts key/value pair into AidaDb according to arguments:
<key> <value>
`,
}

// insert given key/value pair into AidaDb
func insert(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	if ctx.Args().Len() != 2 {
		return fmt.Errorf("this command requires two arguments - <key> <value>")
	}

	key := ctx.Args().Get(0)
	val := ctx.Args().Get(1)

	// open aidaDb
	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb. Error: %v", err)
	}

	defer MustCloseDB(aidaDb)

	value, err := rlp.EncodeToBytes(val)
	if err != nil {
		return fmt.Errorf("cannot encode given value %v; %v", val, err)
	}

	return aidaDb.Put([]byte(key), value)
}
