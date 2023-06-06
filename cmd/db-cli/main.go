package main

import (
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/db-cli/db"

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
		&db.GenerateCommand,
		&db.MergeCommand,
		&db.UpdateCommand,
		&db.InfoCommand,
	},
}

// main implements aida-db functions
func main() {
	if err := InitDb.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
