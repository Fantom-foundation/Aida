// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/aida-sdb/trace"
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
			&RecordCommand,
			&trace.TraceReplayCommand,
			&trace.TraceReplaySubstateCommand,
		},
	}
}

// main implements "trace" cli traceApplication.
func main() {
	app := initTraceApp()
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
