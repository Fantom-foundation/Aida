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

package flags

import "github.com/urfave/cli/v2"

var (
	Account = cli.StringFlag{
		Name:    "account",
		Usage:   "Wanted account",
		Aliases: []string{"a"},
	}
	Detailed = cli.BoolFlag{
		Name:    "detailed",
		Usage:   "Prints detailed info with how many records is in each prefix",
		Aliases: []string{"d"},
	}
	SkipMetadata = cli.BoolFlag{
		Name:  "skip-metadata",
		Usage: "Skips metadata inserting and getting. Useful especially when working with old AidaDb that does not have Metadata yet",
	}
	InsertFlag = cli.BoolFlag{
		Name:  "insert",
		Usage: "Inserts printed db-hash into AidaDb",
	}
	ForceFlag = cli.BoolFlag{
		Name:  "force",
		Usage: "Forces generation even when dbHash is found.",
	}
)
