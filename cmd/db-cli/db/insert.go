package db

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/flags"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
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
		&flags.EncodingType,
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

	var value []byte

	switch ctx.String(flags.EncodingType.Name) {
	case "":
		return fmt.Errorf("choose encoding type (--encoding-type: byte/rlp/uint/block)")
	case "byte":
		value = []byte(val)
	case "rlp":
		value, err = rlp.EncodeToBytes(val)
		if err != nil {
			return fmt.Errorf("cannot encode given value %v; %v", val, err)
		}
	case "uint":
		u, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", val, err)
		}
		binary.BigEndian.PutUint64(value, u)
	case "block":
		u, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint %v; %v", val, err)
		}
		value = substate.BlockToBytes(u)
	default:
		return fmt.Errorf("unknown encoding type (--encoding-type: byte/rlp/uint/block)")
	}

	// open aidaDb
	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open targetDb. Error: %v", err)
	}

	defer MustCloseDB(aidaDb)

	return aidaDb.Put([]byte(key), value)
}
