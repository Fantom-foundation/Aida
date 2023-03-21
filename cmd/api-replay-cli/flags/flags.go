package flags

import "github.com/urfave/cli/v2"

var (
	// APIRecordingSrcFileFlag defines path to data recorded on API
	APIRecordingSrcFileFlag = cli.PathFlag{
		Name:  "api-recording",
		Usage: "Path to source file with recorded API data",
	}
)
