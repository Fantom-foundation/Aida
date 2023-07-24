package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/substate-cli/replay"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "Substate CLI Manger",
		HelpName:  "substate-cli",
		Copyright: "(c) 2022 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&replay.ReplayCommand,
			&replay.GenDeletedAccountsCommand,
			&replay.GetStorageUpdateSizeCommand,
			&replay.GetCodeCommand,
			&replay.GetCodeSizeCommand,
			&replay.SubstateDumpCommand,
			&replay.GetAddressStatsCommand,
			&replay.GetKeyStatsCommand,
			&replay.GetLocationStatsCommand,
		},
	}
	substate.RecordReplay = true
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
