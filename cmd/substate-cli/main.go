package main

import (
	"fmt"
	"os"

	db2 "github.com/Fantom-foundation/Aida/cmd/substate-cli/db"

	"github.com/Fantom-foundation/Aida/cmd/substate-cli/replay"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

var (
	dbCommand = cli.Command{
		Name:        "db",
		Usage:       "A set of commands on substate DB",
		Description: "",
		Subcommands: []*cli.Command{
			&db2.CloneCommand,
			&db2.CompactCommand,
		},
	}
)

var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate   = ""
)

func main() {
	app := &cli.App{
		Name:      "Substate CLI Manger",
		HelpName:  "substate-cli",
		Version:   params.VersionWithCommit(gitCommit, gitDate),
		Copyright: "(c) 2022 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&replay.ReplayCommand,
			&replay.GetStorageUpdateSizeCommand,
			&replay.GetCodeCommand,
			&replay.GetCodeSizeCommand,
			&replay.SubstateDumpCommand,
			&replay.GetAddressStatsCommand,
			&replay.GetKeyStatsCommand,
			&replay.GetLocationStatsCommand,
			&dbCommand,
		},
	}
	substate.RecordReplay = true
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
