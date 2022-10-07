package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/substate"
	"github.com/Fantom-foundation/aida/cmd/trace"
	"github.com/Fantom-foundation/go-opera/flags"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate   = ""

	app = flags.NewApp(gitCommit, gitDate, "Fantom storage trace command line interface")
)

func init() {
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		trace.TraceRecordCommand,
		trace.TraceReplayCommand,
	}
	cli.CommandHelpTemplate = flags.CommandHelpTemplate
}

func main() {
	substate.RecordReplay = true
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
