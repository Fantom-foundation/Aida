package flags

import "github.com/urfave/cli/v2"

var (
	Skip = cli.Uint64Flag{
		Name:  "skip",
		Usage: "Skips first N requests",
	}
)
