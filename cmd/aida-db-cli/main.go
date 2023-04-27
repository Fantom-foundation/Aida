package main

import (
	"log"
	"os"

	"github.com/Fantom-foundation/Aida/cmd/aida-db-cli/db"

	"github.com/urfave/cli/v2"
)

// InitDb data structure
var InitDb = cli.App{
	Name:      "Aida Database",
	HelpName:  "aida-db",
	Usage:     "merge source data into profiling database",
	Copyright: "(c) 2022 Fantom Foundation",
	Commands: []*cli.Command{
		&db.MergeCommand,
		&db.UpdateCommand,
	},
	Description: `
app for maintaining with aida-db`,
}

// main implements aida-db functions
func main() {
	if err := InitDb.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
