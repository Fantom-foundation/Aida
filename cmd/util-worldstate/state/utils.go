// Package state implements executable entry points to the world state generator app.
package state

import (
	"fmt"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

// DefaultPath provides path set for the given flag, or builds a default path
// adding provided sub-dir to the user's home dir.
func DefaultPath(ctx *cli.Context, flag *cli.PathFlag, def string) (string, error) {
	// do we have an explicit path?
	if ctx.IsSet(flag.Name) {
		pa := ctx.Path(flag.Name)
		if pa != "" {
			return pa, nil
		}
	}

	// obtain user home dir
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home dir; %s", err.Error())
	}

	// apply default path to the context
	err = ctx.Set(flag.Name, path.Join(dir, def))
	if err != nil {
		return "", fmt.Errorf("could not set default DB path; %s", err.Error())
	}

	return ctx.Path(flag.Name), nil
}
