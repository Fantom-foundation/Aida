package db

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
	// stateDbDirFlag defines path to opera state database
	stateDbDirFlag = cli.PathFlag{
		Name:  "state-db-dir",
		Usage: "Path for the opera state database.",
		Value: "",
	}
	// substateDirFlag defines directory to substate database
	substateDirFlag = cli.PathFlag{
		Name:  "substate-db-dir",
		Usage: "Substate database directory.",
		Value: "",
	}
	// dbNameFlag defines database file name
	dbNameFlag = cli.StringFlag{
		Name:  "db-name",
		Usage: "Database name.",
		Value: "main",
	}
	// dbNameFlag defines database file name
	dbTypeFlag = cli.StringFlag{
		Name:  "state-db-type",
		Usage: "Type of database (\"ldb\" or \"pbl\") (default: ldb)",
		Value: "ldb",
	}
	// workersFlag defines number of worker threads that execute in parallel
	workersFlag = cli.IntFlag{
		Name:  "workers",
		Usage: "Number of worker threads that execute in parallel",
		Value: 4,
	}
)
