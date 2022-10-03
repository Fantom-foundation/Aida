package dump

import (
	"github.com/urfave/cli/v2"
)

var (
	// rootHashFlag specifies root of the trie
	rootHashFlag = cli.StringFlag{
		Name:  "root-hash",
		Usage: "Root of trie.",
		Value: "",
	}
	// stateDBFlag defines path to opera state database
	stateDBFlag = cli.PathFlag{
		Name:  "state-dump-dir",
		Usage: "Path for the opera state database.",
		Value: "",
	}
	// outputDBFlag defines directory to account-state database
	outputDBFlag = cli.PathFlag{
		Name:  "accstate-dump-dir",
		Usage: "Substate database directory.",
		Value: "",
	}
	// dbNameFlag defines database file name
	dbNameFlag = cli.StringFlag{
		Name:  "dump-name",
		Usage: "Database name.",
		Value: "main",
	}
	// dbNameFlag defines database file name
	dbTypeFlag = cli.StringFlag{
		Name:  "state-dump-type",
		Usage: "Type of database (\"ldb\" or \"pbl\") (default: ldb)",
		Value: "ldb",
	}
	// workersFlag defines number of handleAccounts threads that execute in parallel
	workersFlag = cli.IntFlag{
		Name:  "workers",
		Usage: "Number of handleAccounts threads that execute in parallel",
		Value: 4,
	}
)
