// Package main defines the World State Manager entry point
package main

import (
	"github.com/Fantom-foundation/Aida-Testing/cmd/gen-world-state/flags"
	"github.com/Fantom-foundation/Aida-Testing/cmd/gen-world-state/state"
	"github.com/Fantom-foundation/Aida-Testing/cmd/gen-world-state/version"
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
			&state.CmdAccount,
			&state.CmdClone,
			&state.CmdCompareState,
			&state.CmdDumpState,
			&state.CmdEvolveState,
			&version.CmdVersion,
		},
		Flags: []cli.Flag{
			&flags.StateDBPath,
			&flags.LogLevel,
		},
		Before:                 assertDBPath,
		UseShortOptionHandling: true,
	}

	// execute the application based on provided arguments
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// assertDBPath makes sure a default world state path is set in the calling flags.
func assertDBPath(ctx *cli.Context) error {
	state.DefaultPath(ctx, &flags.StateDBPath, ".aida/world-state")
	return nil
}
