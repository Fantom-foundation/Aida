package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/aida-vm/vm"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "Substate CLI Manger",
		HelpName:  "aida-vm",
		Copyright: "(c) 2022 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&vm.ReplayCommand,
			&vm.GetCodeSizeCommand,
			&vm.SubstateDumpCommand,
			&vm.GetAddressStatsCommand,
			&vm.GetKeyStatsCommand,
			&vm.GetLocationStatsCommand,
		},
	}
	substate.RecordReplay = true
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
