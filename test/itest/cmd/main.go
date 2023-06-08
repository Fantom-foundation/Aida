package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/rc-testing/test/itest/cmd/stvm"
	"github.com/Fantom-foundation/rc-testing/test/itest/cmd/stvmdb"
	"github.com/urfave/cli/v2"
)

var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate   = ""
)

func main() {
	app := &cli.App{
		Name:     "Integration Tester",
		HelpName: "itest",
		// Version:   params.VersionWithCommit(gitCommit, gitDate),
		Copyright: "(c) 2022 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&stvm.StVmCommand,
			&stvmdb.StVmDbCommand,
		},
	}
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
