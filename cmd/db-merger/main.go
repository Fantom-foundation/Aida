package main

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/db-merger/dbmerger"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/urfave/cli/v2"
)

// DbMergerApp data structure
var DbMergerApp = cli.App{
	Action:    dbmerger.DbMerger,
	Name:      "Aida Database Merger",
	HelpName:  "dbmerger",
	Usage:     "merge source data into profiling database",
	Copyright: "(c) 2022 Fantom Foundation",
	ArgsUsage: "",
	Flags: []cli.Flag{
		&utils.DeleteSourceDbsFlag,
		&utils.AidaDbFlag,
		&utils.LogLevel,
	},
	Description: `
The dbmerger command merges databases with source data into one database which is used for profiling.

dbmerger command requires paths to: substatedir, updatedir, deleted-account-dir`,
}

// main implements dbmerger.
func main() {
	if err := DbMergerApp.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
