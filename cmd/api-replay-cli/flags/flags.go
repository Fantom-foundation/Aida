package flags

import "github.com/urfave/cli/v2"

var (
	// APIRecordingSrcFileFlag defines path to data recorded on API
	APIRecordingSrcFileFlag = cli.PathFlag{
		Name:  "api-recording",
		Usage: "Path to source file with recorded API data",
	}
	// WorkersFlag defines number of threads for execution
	WorkersFlag = cli.IntFlag{
		Name:  "workers",
		Usage: "defines how many number of threads in which request execution into StateDB is run on. Default: 4",
		Value: 4,
	}
)
