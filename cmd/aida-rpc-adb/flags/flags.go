package flags

import "github.com/urfave/cli/v2"

var (
	Skip = cli.Uint64Flag{
		Name:  "skip",
		Usage: "Skips first N requests",
	}
	LogToFile = cli.BoolFlag{
		Name:  "log-to-file",
		Usage: "Logs any data mismatch into a file",
	}
	LogFileDir = cli.PathFlag{
		Name:  "log-file-dir",
		Usage: "Determines the dir when log file will be saved",
		Value: "/var/opera/Aida/logs",
	}
)
