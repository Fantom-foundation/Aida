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

	"github.com/Fantom-foundation/Aida/cmd/aida-profile/profile"
	"github.com/urfave/cli/v2"
)

// main implements aida-profile cli.
func main() {
	app := cli.App{
		Name:      "Aida Storage Profile Manager",
		HelpName:  "profile",
		Usage:     "profile on the world-state",
		Copyright: "(c) 2023 Fantom Foundation",
		Commands: []*cli.Command{
			&profile.GetCodeSizeCommand,
			&profile.GetStorageUpdateSizeCommand,
			&profile.GetAddressStatsCommand,
			&profile.GetKeyStatsCommand,
			&profile.GetLocationStatsCommand,
		},
	}
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
