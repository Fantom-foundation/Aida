package root

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/db/opera"
	"github.com/Fantom-foundation/Aida-Testing/world-state/logger"
	"github.com/urfave/cli/v2"
)

const (
	flagInputDBPath = "db"
	flagInputDBType = "input-db-type"
	flagStateDBName = "input-db-name"
	flagBlock       = "block"
)

// CmdRoot retrieves root hash for given block number
var CmdRoot = cli.Command{
	Action:      root,
	Name:        "root",
	Aliases:     []string{"r"},
	Usage:       "Retrieve root hash of given block number",
	Description: `Searches opera database for root hash for supplied block number.`,
	ArgsUsage:   "<to>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  flagStateDBName,
			Usage: "Input state trie database table name",
			Value: "main",
		},
		&cli.StringFlag{
			Name:  flagInputDBType,
			Usage: "Type of input database (\"ldb\" or \"pbl\")",
			Value: "ldb",
		},
		&cli.Uint64Flag{
			Name:     flagBlock,
			Usage:    "Block number of desired root",
			Required: true,
		},
	},
}

// root retrieves root hash of given block number
func root(ctx *cli.Context) error {
	// open the source trie DB
	store, err := opera.Connect(ctx.String(flagInputDBType), ctx.Path(flagInputDBPath), ctx.Path(flagStateDBName))
	if err != nil {
		return err
	}
	defer opera.MustCloseStore(store)

	// make logger
	log := logger.New(ctx.App.Writer, "info")

	//look up root hash from block number
	root, err := opera.RootOfBLock(store, ctx.Uint64(flagBlock))
	if err != nil {
		log.Errorf("Unable to find root hash for block number %d; %s", ctx.Uint64(flagBlock), err.Error())
		return err
	}

	log.Infof("Block %d has root hash %s", ctx.Uint64(flagBlock), root)
	log.Info("done")
	return nil
}
