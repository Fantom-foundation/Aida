// Package main defines the World State Manager entry point
package main

import (
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/cmd/build"
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/cmd/db"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

// main implements World State CLI application entry point
func main() {
	// prep the application, pull in all the available command
	app := &cli.App{
		Name:      "Aida World State Manager",
		HelpName:  "worldstate-cli",
		Usage:     "creates and manages copy of EVM world state for off-the-chain testing and profiling",
		Copyright: "(c) 2022 Fantom Foundation",
		Version:   build.Version,
		Commands: []*cli.Command{
			&build.CmdVersion,
			&db.StateDumpCommand,
		},
	}

	// execute the application based on provided arguments
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
