// Package main defines the World State Manager entry point
package main

import (
	"github.com/Fantom-foundation/Aida-Testing/cmd/gen-world-state/version"
	"github.com/Fantom-foundation/Aida-Testing/world-state/account"
	"github.com/Fantom-foundation/Aida-Testing/world-state/clone"
	"github.com/Fantom-foundation/Aida-Testing/world-state/compare"
	"github.com/Fantom-foundation/Aida-Testing/world-state/dump"
	"github.com/Fantom-foundation/Aida-Testing/world-state/evolve"
	"github.com/Fantom-foundation/Aida-Testing/world-state/root"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

// main implements World State CLI application entry point
func main() {
	// prep the application, pull in all the available command
	app := &cli.App{
		Name:      "Aida World State Manager",
		HelpName:  "gen-world-state",
		Usage:     "creates and manages copy of EVM world state for off-the-chain testing and profiling",
		Copyright: "(c) 2022 Fantom Foundation",
		Version:   version.Version,
		Commands: []*cli.Command{
			&dump.CmdDumpState,
			&evolve.CmdEvolveState,
			&account.CmdAccount,
			&clone.CmdClone,
			&compare.CmdCompareState,
			&root.CmdRoot,
			&version.CmdVersion,
		},
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "db",
				Usage:    "World state snapshot database path.",
				Value:    "",
				Required: true,
			},
		},
	}

	// execute the application based on provided arguments
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
