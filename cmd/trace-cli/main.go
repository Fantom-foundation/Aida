package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/trace-cli/trace"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// initTraceApp initializes a trace-cli app. This function is called by the main
// function and unit tests.
func initTraceApp() *cli.App {
	return &cli.App{
		Name:      "Aida Storage Trace Manager",
		HelpName:  "trace",
		Copyright: "(c) 2022 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&trace.TraceCompareLogCommand,
			&trace.TraceRecordCommand,
			&trace.TraceReplayCommand,
			&trace.TraceReplaySubstateCommand,
		},
	}
}

// main implements "trace" cli traceApplication.
func main() {
	substate.RecordReplay = true
	app := initTraceApp()
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
