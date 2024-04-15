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

	"github.com/Fantom-foundation/Aida/cmd/util-updateset/updateset"
	"github.com/urfave/cli/v2"
)

// GenUpdateSetApp data structure
var GenUpdateSetApp = cli.App{
	Name:      "Aida Generate Update-set Manager",
	HelpName:  "aida-updateset",
	Usage:     "generate update-set from substate",
	Copyright: "(c) 2022 Fantom Foundation",
	ArgsUsage: "<blockNumLast> <interval>",
	Flags:     []cli.Flag{},
	Commands: []*cli.Command{
		&updateset.GenUpdateSetCommand,
		&updateset.UpdateSetStatsCommand,
	},
}

// main implements gen-update-set cli.
func main() {
	if err := GenUpdateSetApp.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
