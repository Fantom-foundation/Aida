// Package flags defines all the flags used by the world state generator app.
package flags

import "github.com/urfave/cli/v2"

var (
	// TargetBlock represents the ID of target block to be reached by state evolve process or in dump state
	TargetBlock = cli.Uint64Flag{
		Name:    "block",
		Aliases: []string{"block", "blk"},
		Usage:   "target block ID",
		Value:   0,
	}
)
