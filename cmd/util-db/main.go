package main

import (
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/util-db/db"

	"github.com/urfave/cli/v2"
)

// UtilDbApp data structure
var UtilDbApp = cli.App{
	Name:      "Aida Database",
	HelpName:  "util-db",
	Usage:     "merge source data into profiling database",
	Copyright: "(c) 2022 Fantom Foundation",
	Commands: []*cli.Command{
		&db.AutoGenCommand,
		&db.CloneCommand,
		&db.CompactCommand,
		&db.GenerateCommand,
		&db.ExtractEthereumGenesisCommand,
		&db.LachesisUpdateCommand,
		&db.MergeCommand,
		&db.UpdateCommand,
		&db.InfoCommand,
		&db.ValidateCommand,
		&db.GenDeletedAccountsCommand,
		&db.SubstateDumpCommand,
		&db.GenerateDbHashCommand,
		&db.PrintDbHashCommand,
		&db.PrintPrefixHashCommand,
		&db.PrintTableHashCommand,
		&db.ScrapeCommand,
		&db.MetadataCommand,
	},
}

// main implements aida-db functions
func main() {
	if err := UtilDbApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
