// Package flags defines all the flags used by the world state generator app.
package flags

import "github.com/urfave/cli/v2"

var (
	// StateDBPath defines the path to the world state snapshot database
	StateDBPath = cli.PathFlag{
		Name:  "db",
		Usage: "World state snapshot database path.",
	}

	// SourceDBPath represents the path of a source DB
	SourceDBPath = cli.PathFlag{
		Name:  "from",
		Usage: "source database path",
	}

	// SourceDBType represents the type of source database
	SourceDBType = cli.StringFlag{
		Name:  "db-type",
		Usage: "type of the source database (\"ldb\" or \"pbl\")",
		Value: "ldb",
	}

	// SourceTableName represents the name of a source DB table
	SourceTableName = cli.StringFlag{
		Name:  "table",
		Usage: "name of the database table to be used",
		Value: "main",
	}

	// TargetDBPath represents the path of a target DB
	TargetDBPath = cli.PathFlag{
		Name:  "to",
		Usage: "target database path",
	}

	// TrieRootHash represents a hash of a state trie root to be decoded
	TrieRootHash = cli.StringFlag{
		Name:  "root",
		Usage: "state trie root hash to be analysed",
	}

	// Workers represents a number of worker threads to be used for computation
	Workers = cli.IntFlag{
		Name:  "workers",
		Usage: "number of worker threads to be used",
		Value: 5,
	}

	// IsStorageIncluded represents a flag for contract storage inclusion in an operation
	IsStorageIncluded = cli.BoolFlag{
		Name:  "with-storage",
		Usage: "display full storage content",
	}
)
