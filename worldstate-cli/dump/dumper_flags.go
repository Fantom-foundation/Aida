package dump

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// Config represents parsed arguments
type Config struct {
	operaStateDBDir  string
	outputDBDir      string
	operaStateDBName string
	rootHash         common.Hash
	dbType           string
	workers          int
}

var (
	// RootHashFlag specifies root of the trie
	RootHashFlag = cli.StringFlag{
		Name:  "root",
		Usage: "Root hash of the state trie",
		Value: "",
	}
	// StateDBFlag defines path to opera state database
	StateDBFlag = cli.PathFlag{
		Name:  "input-db",
		Usage: "Input state database path",
		Value: "",
	}
	// OutputDBFlag defines directory to account-state database
	OutputDBFlag = cli.PathFlag{
		Name:  "output-db",
		Usage: "Output state snapshot database path",
		Value: "",
	}
	// DbNameFlag defines database file name
	DbNameFlag = cli.StringFlag{
		Name:  "input-db-name",
		Usage: "Input state database name",
		Value: "main",
	}
	// DbNameFlag defines database file name
	DbTypeFlag = cli.StringFlag{
		Name:  "input-db-type",
		Usage: "Type of input database (\"ldb\" or \"pbl\")",
		Value: "ldb",
	}
	// WorkersFlag defines number of handleAccounts threads that execute in parallel
	WorkersFlag = cli.IntFlag{
		Name:  "workers",
		Usage: "Number of account processing threads",
		Value: 4,
	}
)

// parseArguments parse arguments into Config
func parseArguments(ctx *cli.Context) *Config {
	// check whether supplied rootHash is not empty
	rootHash := common.HexToHash(ctx.String(RootHashFlag.Name))
	if rootHash == emptyHash {
		log.Panicln("Root hash is not defined.")
	}

	return &Config{
		rootHash:         rootHash,
		operaStateDBDir:  ctx.Path(StateDBFlag.Name),
		outputDBDir:      ctx.Path(OutputDBFlag.Name),
		operaStateDBName: ctx.String(DbNameFlag.Name),
		dbType:           ctx.String(DbTypeFlag.Name),
		workers:          ctx.Int(WorkersFlag.Name)}
}
