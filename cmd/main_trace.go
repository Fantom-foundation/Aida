package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida-Testing/cmd/trace"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
)

// Implement "trace" cli application
func main() {
	app := &cli.App{
		Name:      "Aida Storage Trace Manager",
		HelpName:  "trace",
		Copyright: "(c) 2022 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&trace.TraceRecordCommand,
			&trace.TraceReplayCommand,
		},
	}
	substate.RecordReplay = true
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
