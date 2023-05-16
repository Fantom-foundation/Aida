package flags

import "github.com/urfave/cli/v2"

var (
	Account = cli.StringFlag{
		Name:    "account",
		Usage:   "Wanted account",
		Aliases: []string{"a"},
	}
)
