package flags

import "github.com/urfave/cli/v2"

var (
	Account = cli.StringFlag{
		Name:    "account",
		Usage:   "Wanted account",
		Aliases: []string{"a"},
	}
	Detailed = cli.BoolFlag{
		Name:    "detailed",
		Usage:   "Prints detailed info with how many records is in each prefix",
		Aliases: []string{"d"},
	}
	SkipMetadata = cli.BoolFlag{
		Name:  "skip-metadata",
		Usage: "Skips metadata inserting and getting. Useful especially when working with old AidaDb that does not have Metadata yet",
	}

	InsertFlag = cli.BoolFlag{
		Name:  "insert",
		Usage: "Inserts printed db-hash into AidaDb",
	}
)
