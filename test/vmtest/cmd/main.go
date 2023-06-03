package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/rc-testing/test/vmtest/replay"
	"github.com/Fantom-foundation/rc-testing/test/vmtest/runvm"
	"github.com/urfave/cli/v2"
)

var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate   = ""
)

func main() {
	app := &cli.App{
		Name:     "VM Tester",
		HelpName: "vmtest",
		// Version:   params.VersionWithCommit(gitCommit, gitDate),
		Copyright: "(c) 2022 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&replay.ReplayCommand,
			&runvm.RunVMCommand,
		},
	}
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
