// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/Fantom-foundation/Aida/cmd/util-primer/db"
)

// UtilPrimerApp data structure
var UtilPrimerApp = cli.App{
	Name:      "Aida Database Priming",
	HelpName:  "util-primer",
	Usage:     "priming without block processing",
	Copyright: "(c) 2024 Fantom Foundation",
	Commands: []*cli.Command{
		&db.RunPrimerCmd,
	},
}

// main implements util-primer functions
func main() {
	if err := UtilPrimerApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
