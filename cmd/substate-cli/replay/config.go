package replay

import (
	"github.com/urfave/cli/v2"
)

// chain id
var chainID int
var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate   = ""
)

// command line options
var (
	UseInMemoryStateDbFlag = cli.BoolFlag{
		Name:  "faststatedb",
		Usage: "enables a faster, yet still experimental StateDB implementation",
	}
)
