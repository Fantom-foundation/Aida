package main

import (
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/util-db/db"

	"github.com/urfave/cli/v2"
)

// InitDb data structure
var InitDb = cli.App{
	Name:      "Aida Database",
	HelpName:  "aida-db",
	Usage:     "merge source data into profiling database",
	Copyright: "(c) 2022 Fantom Foundation",
	Commands: []*cli.Command{
		&db.AutoGenCommand,
		&db.CloneCommand,
		&db.CompactCommand,
		&db.GenerateCommand,
		&db.LachesisUpdateCommand,
		&db.MergeCommand,
		&db.UpdateCommand,
		&db.InfoCommand,
		&db.InsertMetadataCommand,
		&db.RemoveMetadataCommand,
		&db.ValidateCommand,
		&db.GenDeletedAccountsCommand,
	},
}

// main implements aida-db functions
func main() {
	if err := InitDb.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
