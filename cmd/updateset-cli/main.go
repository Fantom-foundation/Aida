package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/updateset-cli/updateset"
	"github.com/urfave/cli/v2"
)

// GenUpdateSetApp data structure
var GenUpdateSetApp = cli.App{
	Name:      "Aida Generate Update-set Manager",
	HelpName:  "aida-updateset",
	Usage:     "generate update-set from substate",
	Copyright: "(c) 2022 Fantom Foundation",
	ArgsUsage: "<blockNumLast> <interval>",
	Flags:     []cli.Flag{},
	Commands: []*cli.Command{
		&updateset.GenUpdateSetCommand,
		&updateset.UpdateSetStatsCommand,
	},
}

// main implements gen-update-set cli.
func main() {
	if err := GenUpdateSetApp.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
