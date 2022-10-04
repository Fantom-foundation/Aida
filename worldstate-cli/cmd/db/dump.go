package db

import (
	"github.com/Fantom-foundation/Aida-Testing/worldstate-cli/dump"
	"github.com/urfave/cli/v2"
)

// StateDumpCommand command
var StateDumpCommand = cli.Command{
	Action:    dump.StateDumpAction,
	Name:      "state-dump",
	Usage:     "Dumps state of mpt tree at given root from input database into state snapshot database.",
	ArgsUsage: "<root> <input-db> <output-db> <input-db-name> <input-db-type> <workers>",
	Flags: []cli.Flag{
		&dump.RootHashFlag,
		&dump.StateDBFlag,
		&dump.OutputDBFlag,
		&dump.DbNameFlag,
		&dump.DbTypeFlag,
		&dump.WorkersFlag,
	},
	Description: `
	The worldstate-cli state-dump command requires three arguments:
		<root> containing root hash of the state trie  
		<input-db> path to input database
		<output-db> path to output database`,
}
