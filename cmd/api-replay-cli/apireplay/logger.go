package apireplay

import (
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// defaultLogFormat defines the format used for log output.
const defaultLogFormat = "%{color}%{level:-8s} %{shortpkg}/%{shortfunc}%{color:reset}: %{message}"

// newLogger returns new instance of Logger
func newLogger(ctx *cli.Context, cfg *utils.Config) *logging.Logger {
	backend := logging.NewLogBackend(ctx.App.Writer, "", 0)

	fm := logging.MustStringFormatter(defaultLogFormat)
	fmtBackend := logging.NewBackendFormatter(backend, fm)

	lvl, err := logging.LogLevel(cfg.LogLevel)
	if err != nil {
		lvl = logging.NOTICE
	}
	lvlBackend := logging.AddModuleLevel(fmtBackend)
	lvlBackend.SetLevel(lvl, "")

	logging.SetBackend(lvlBackend)
	return logging.MustGetLogger("api-replay")
}
