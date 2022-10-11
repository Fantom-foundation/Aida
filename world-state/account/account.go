// Package account implements information providers for individual accounts in the state dump database.
package account

import "github.com/urfave/cli/v2"

const (
	flagAccountAddress = "addr"
)

// CmdAccount defines a CLI command set for managing single account data in the state dump database.
var CmdAccount = cli.Command{
	Name:    "account",
	Aliases: []string{"a"},
	Usage:   `Provides information and management function for individual accounts in state dump database.`,
	Subcommands: []*cli.Command{
		&cmdAccountInfo,
	},
}
