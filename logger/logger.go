package logger

//go:generate mockgen -source logger.go -destination logger_mocks.go -package logger

import (
	"os"
	"time"

	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

var LogLevelFlag = cli.StringFlag{
	Name:    "log",
	Aliases: []string{"l"},
	Usage:   "Level of the logging of the app action (\"critical\", \"error\", \"warning\", \"notice\", \"info\", \"debug\"; default: INFO)",
	Value:   "info",
}

// defaultLogFormat defines the format used for log output.
const (
	defaultLogFormat = "%{time:2006/01/02 15:04:05} %{color}%{level:-8s} %{shortpkg}/%{shortfunc}%{color:reset}: %{message}"
)

// Logger is responsible for logging any info to user.
// Generally avoid Fatal and Panic since these two end the program ungracefully.
// Critical should be used a potential unexpected behaviour that could lead to fatal state.
// Error should be used to print any error state.
// Warning should be used for a potential unexpected behaviour, though not fatal.
// Notice should be used to inform user about a milestone.
// Info should be used for repeated messages (reached block 1000 etc...).
type Logger interface {
	// Fatal is equivalent to l.Critical(fmt.Sprint()) followed by a call to os.Exit(1).
	Fatal(args ...interface{})
	// Fatalf is equivalent to l.Critical followed by a call to os.Exit(1).
	Fatalf(format string, args ...interface{})

	// Panic is equivalent to l.Critical(fmt.Sprint()) followed by a call to panic().
	Panic(args ...interface{})
	// Panicf is equivalent to l.Critical followed by a call to panic().
	Panicf(format string, args ...interface{})

	// Critical logs a message using CRITICAL as log level.
	Critical(args ...interface{})
	// Criticalf logs a message using CRITICAL as log level.
	Criticalf(format string, args ...interface{})

	// Error logs a message using ERROR as log level.
	Error(args ...interface{})
	// Errorf logs a message using ERROR as log level.
	Errorf(format string, args ...interface{})

	// Warning logs a message using WARNING as log level.
	Warning(args ...interface{})
	// Warningf logs a message using WARNING as log level.
	Warningf(format string, args ...interface{})

	// Notice logs a message using NOTICE as log level.
	Notice(args ...interface{})
	// Noticef logs a message using NOTICE as log level.
	Noticef(format string, args ...interface{})

	// Info logs a message using INFO as log level.
	Info(args ...interface{})
	// Infof logs a message using INFO as log level.
	Infof(format string, args ...interface{})

	// Debug logs a message using DEBUG as log level.
	Debug(args ...interface{})
	// Debugf logs a message using DEBUG as log level.
	Debugf(format string, args ...interface{})
}

// NewLogger provides a new instance of the Logger based on context flags.
func NewLogger(level string, module string) *logging.Logger {
	backend := logging.NewLogBackend(os.Stdout, "", 0)

	fm := logging.MustStringFormatter(defaultLogFormat)
	fmtBackend := logging.NewBackendFormatter(backend, fm)

	lvl, err := logging.LogLevel(level)
	if err != nil {
		lvl = logging.INFO
	}
	lvlBackend := logging.AddModuleLevel(fmtBackend)
	lvlBackend.SetLevel(lvl, "")

	logging.SetBackend(lvlBackend)
	return logging.MustGetLogger(module)
}

// ParseTime from seconds to hours, minutes and seconds
func ParseTime(elapsed time.Duration) (uint32, uint32, uint32) {
	var (
		hours, minutes, seconds uint32
	)

	seconds = uint32(elapsed.Round(1 * time.Second).Seconds())

	if seconds > 60 {
		minutes = seconds / 60
		seconds -= minutes * 60
	}

	if minutes > 60 {
		hours = minutes / 60
		minutes -= hours * 60
	}

	return hours, minutes, seconds
}
