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
		Aliases: []string{"a"},
	}
	EncodingType = cli.StringFlag{
		Name:     "encoding-type",
		Usage:    "Choose encoding for value when inserting into AidaDb (uint, byte, rlp)",
		Required: true,
	}
)
