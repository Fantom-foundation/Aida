package db

import (
	"github.com/urfave/cli/v2"
)

// rootHashFlag specifies root of the trie
var rootHashFlag = cli.StringFlag{
	Name:  "root-hash",
	Usage: "Root of trie.",
	Value: "",
}

// dbDirFlag defines directory to store Lachesis state and user's wallets
var dbDirFlag = cli.PathFlag{
	Name:  "db-dir",
	Usage: "Data directory for the database.",
	Value: "",
}

// dbNameFlag defines database file name
var dbNameFlag = cli.StringFlag{
	Name:  "db-name",
	Usage: "Database name.",
	Value: "main",
}

// dbNameFlag defines database file name
var dbTypeFlag = cli.StringFlag{
	Name:  "db-type",
	Usage: "Type of database (\"ldb\" or \"pbl\") (default: ldb)",
	Value: "ldb",
}
