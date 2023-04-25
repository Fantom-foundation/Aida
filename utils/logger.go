package utils

import (
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// defaultLogFormat defines the format used for log output.
const defaultLogFormat = "%{color}%{level:-8s} %{shortpkg}/%{shortfunc}%{color:reset}: %{message}"

// NewLogger provides a new instance of the Logger based on context flags.
func NewLogger(ctx *cli.Context, module string) *logging.Logger {
	backend := logging.NewLogBackend(ctx.App.Writer, "", 0)

	fm := logging.MustStringFormatter(defaultLogFormat)
	fmtBackend := logging.NewBackendFormatter(backend, fm)

	lvl, err := logging.LogLevel(ctx.String(LogLevel.Name))
	if err != nil {
		lvl = logging.INFO
	}
	lvlBackend := logging.AddModuleLevel(fmtBackend)
	lvlBackend.SetLevel(lvl, "")

	logging.SetBackend(lvlBackend)
	return logging.MustGetLogger(module)
}
