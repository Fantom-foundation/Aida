package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/op/go-logging"
)

// defaultLogFormat defines the format used for log output.
const (
	defaultLogFormat = "%{color}%{level:-8s} %{shortpkg}/%{shortfunc}%{color:reset}: %{message}"
	timestampFormat  = "2006/01/02 15:04:05"
)

// NewLogger provides a new instance of the Logger based on context flags.
func NewLogger(level string, module string) *logging.Logger {
	prefix := fmt.Sprintf("%v\t", time.Now().Format(timestampFormat))

	backend := logging.NewLogBackend(os.Stdout, prefix, 0)

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
