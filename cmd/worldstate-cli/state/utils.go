// Package state implements executable entry points to the world state generator app.
package state

import (
	"fmt"
	"os"
	"path"

	"github.com/Fantom-foundation/Aida/cmd/worldstate-cli/flags"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// defaultLogFormat defines the format used for log output.
const defaultLogFormat = "%{color}%{level:-8s} %{shortpkg}/%{shortfunc}%{color:reset}: %{message}"

// DefaultPath provides path set for the given flag, or builds a default path
// adding provided sub-dir to the user's home dir.
func DefaultPath(ctx *cli.Context, flag *cli.PathFlag, def string) string {
	// do we have an explicit path?
	if ctx.IsSet(flag.Name) {
		pa := ctx.Path(flag.Name)
		if pa != "" {
			return pa
		}
	}

	// obtain user home dir
	dir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("could not get user home dir; %s", err.Error()))
	}

	// apply default path to the context
	err = ctx.Set(flags.StateDBPath.Name, path.Join(dir, def))
	if err != nil {
		panic(fmt.Errorf("could not set default DB path; %s", err.Error()))
	}

	return ctx.Path(flag.Name)
}

// Logger provides a new instance of the Logger based on context flags.
func Logger(ctx *cli.Context, module string) *logging.Logger {
	backend := logging.NewLogBackend(ctx.App.Writer, "", 0)

	fm := logging.MustStringFormatter(defaultLogFormat)
	fmtBackend := logging.NewBackendFormatter(backend, fm)

	lvl, err := logging.LogLevel(ctx.String(flags.LogLevel.Name))
	if err != nil {
		lvl = logging.INFO
	}
	lvlBackend := logging.AddModuleLevel(fmtBackend)
	lvlBackend.SetLevel(lvl, "")

	logging.SetBackend(lvlBackend)
	return logging.MustGetLogger(module)
}
