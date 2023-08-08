package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/aida-profile/parallelisation"
	"github.com/Fantom-foundation/Aida/cmd/aida-profile/profile"
	"github.com/urfave/cli/v2"
)

// main implements aida-profile cli.
func main() {
	app := cli.App{
		Name:      "Aida Storage Profile Manager",
		HelpName:  "profile",
		Usage:     "profile on the world-state",
		Copyright: "(c) 2023 Fantom Foundation",
		Commands: []*cli.Command{
			&parallelisation.ParallelisationCommand,

			// profile
			&profile.GetStorageUpdateSizeCommand,
		},
	}
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
