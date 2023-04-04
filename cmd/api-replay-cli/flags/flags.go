package flags

import "github.com/urfave/cli/v2"

var (
	// APIRecordingSrcFileFlag defines path to data recorded on API
	APIRecordingSrcFileFlag = cli.PathFlag{
		Name:    "api-recording",
		Usage:   "Path to source file with recorded API data",
		Aliases: []string{"r"},
	}
	// WorkersFlag defines number of threads for execution
	WorkersFlag = cli.IntFlag{
		Name: "workers",
		Usage: "defines the thread number for api-replay. " +
			"The exact value is used for number of Executor threads, " +
			"number of Comparator threads is the number divided by 2 since the Execution is much slower;" +
			"default: 4",
		Value:   4,
		Aliases: []string{"w"},
	}
	// LogLevel defines the level of logging of the app
	LogLevel = cli.StringFlag{
		Name:    "log",
		Aliases: []string{"l"},
		Usage:   "Level of the logging of the app action (\"critical\", \"error\", \"warning\", \"notice\", \"info\", \"debug\"; default: INFO)",
		Value:   "info",
	}
	Verbose = cli.BoolFlag{
		Name:    "verbose",
		Usage:   "defines that we want to log steps in replay",
		Aliases: []string{"v"},
	}
	ContinueOnFailure = cli.BoolFlag{
		Name:    "continue-on-failure",
		Usage:   "if used, any mismatched data will not exit the program",
		Aliases: []string{"cof"},
	}
	// WorkersFlag defines number of threads for execution
	WorkersFlag = cli.IntFlag{
		Name: "workers",
		Usage: "defines the thread number for api-replay. " +
			"The exact value is used for number of Executor threads, " +
			"number of Comparator threads is the number divided by 2 since the Execution is much slower;" +
			"default: 4",
		Value: 4,
	}
	// LogLevel defines the level of logging of the app
	LogLevel = cli.StringFlag{
		Name:    "log",
		Aliases: []string{"l"},
		Usage:   "Level of the logging of the app action (\"critical\", \"error\", \"warning\", \"notice\", \"info\", \"debug\"; default: INFO)",
		Value:   "info",
	}
	Verbose = cli.BoolFlag{
		Name:  "verbose",
		Usage: "defines that we want to log steps in replay",
	}
)
