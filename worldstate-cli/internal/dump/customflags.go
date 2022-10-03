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
	// rootHashFlag specifies root of the trie
	rootHashFlag = cli.StringFlag{
		Name:  "root",
		Usage: "Root hash of the state trie.",
		Value: "",
	}
	// stateDBFlag defines path to opera state database
	stateDBFlag = cli.PathFlag{
		Name:  "input-db",
		Usage: "Input state database path.",
		Value: "",
	}
	// outputDBFlag defines directory to account-state database
	outputDBFlag = cli.PathFlag{
		Name:  "output-db",
		Usage: "Output state snapshot database path.",
		Value: "",
	}
	// dbNameFlag defines database file name
	dbNameFlag = cli.StringFlag{
		Name:  "input-db-name",
		Usage: "Input state database name. (default: main)",
		Value: "main",
	}
	// dbNameFlag defines database file name
	dbTypeFlag = cli.StringFlag{
		Name:  "input-db-type",
		Usage: "Type of input database, (\"ldb\" or \"pbl\") (default: ldb)",
		Value: "ldb",
	}
	// workersFlag defines number of handleAccounts threads that execute in parallel
	workersFlag = cli.IntFlag{
		Name:  "workers",
		Usage: "Number of account processing threads. (default: 4)",
		Value: 4,
	}
)

// parseArguments parse arguments into Config
func parseArguments(ctx *cli.Context) *Config {
	// check whether supplied rootHash is not empty
	rootHash := common.HexToHash(ctx.String(rootHashFlag.Name))
	if rootHash == emptyHash {
		log.Panicln("Root hash is not defined.")
	}

	return &Config{
		rootHash:         rootHash,
		operaStateDBDir:  ctx.Path(stateDBFlag.Name),
		outputDBDir:      ctx.Path(outputDBFlag.Name),
		operaStateDBName: ctx.String(dbNameFlag.Name),
		dbType:           ctx.String(dbTypeFlag.Name),
		workers:          ctx.Int(workersFlag.Name)}
}
