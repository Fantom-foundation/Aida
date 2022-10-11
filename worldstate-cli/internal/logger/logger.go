package logger

import (
	"github.com/op/go-logging"
	"log"
	"os"
	"strings"
)

// CliLogger defines extended logger with generic no-level logging option
type CliLogger struct {
	log *log.Logger
	*logging.Logger
}

// New provides pre-configured Logger with stderr output and leveled filtering.
// Modules are not supported at the moment, but may be added in the future to make the logging setup more granular.
func New() Logger {
	backend := logging.NewLogBackend(os.Stderr, "", 0)

	// Parse log format from configuration and apply it to the backend
	format := logging.MustStringFormatter("%{color}%{level:-8s} %{module}%{color:reset}: %{message}")
	fmtBackend := logging.NewBackendFormatter(backend, format)

	level := logging.INFO

	lvlBackend := logging.AddModuleLevel(fmtBackend)
	lvlBackend.SetLevel(level, "")

	// assign the backend and return the new logger
	logging.SetBackend(lvlBackend)
	l := logging.MustGetLogger("WorldState-cli")

	al := CliLogger{
		//log:    log.New(nil, "worldstate-cli", 0),
		Logger: l,
	}

	return &al
}

// ModuleName returns the name of the logger module.
func (al *CliLogger) ModuleName() string {
	return al.Module
}

// Printf implements default non-leveled output.
// We assume the information is low in importance if passed to this function, so we relay it to Debug level.
func (al *CliLogger) Printf(format string, args ...interface{}) {
	al.Debugf(format, args...)
}

// ModuleLogger derives new logger for sub-module.
func (al *CliLogger) ModuleLogger(mod string) Logger {
	var sb strings.Builder
	sb.WriteString(al.Module)
	sb.WriteString(".")
	sb.WriteString(mod)

	l := logging.MustGetLogger(sb.String())
	return &CliLogger{Logger: l, log: al.log}
}

// Log returns log.Logger compatible logging instance.
func (al *CliLogger) Log() *log.Logger {
	return al.log
}
