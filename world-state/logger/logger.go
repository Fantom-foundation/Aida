// Package logger implements well formatted and decorated logging facility.
package logger

import (
	"github.com/op/go-logging"
	"io"
)

// defaultLogFormat defines the format used for log output.
const defaultLogFormat = "%{color}%{level:-8s} %{shortpkg}/%{shortfunc}%{color:reset}: %{message}"

// New provides a new logger for the given output writer.
func New(out io.Writer, level string) *logging.Logger {
	backend := logging.NewLogBackend(out, "", 0)

	fmt := logging.MustStringFormatter(defaultLogFormat)
	fmtBackend := logging.NewBackendFormatter(backend, fmt)

	lvl, err := logging.LogLevel(level)
	if err != nil {
		lvl = logging.INFO
	}
	lvlBackend := logging.AddModuleLevel(fmtBackend)
	lvlBackend.SetLevel(lvl, "")

	logging.SetBackend(lvlBackend)
	return logging.MustGetLogger("")
}
