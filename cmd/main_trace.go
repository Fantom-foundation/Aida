package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/aida/cmd/trace"
	"github.com/Fantom-foundation/go-opera/flags"
	"github.com/ethereum/go-ethereum/substate"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate   = ""

	app = flags.NewApp(gitCommit, gitDate, "Fantom storage trace command line interface")
)

// inits configures flags and sub-commands of trace cli
func init() {
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		trace.TraceRecordCommand,
		trace.TraceReplayCommand,
	}
	cli.CommandHelpTemplate = flags.CommandHelpTemplate
}

// main implements "trace" cli application entry point
func main() {
	substate.RecordReplay = true
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
