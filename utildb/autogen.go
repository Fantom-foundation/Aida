// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utildb

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/urfave/cli/v2"
)

// SetLock creates lockfile in case of error while generating
func SetLock(cfg *utils.Config, message string) error {
	lockFile := cfg.AidaDb + ".autogen.lock"

	// Write the string to the file
	err := os.WriteFile(lockFile, []byte(message), 0655)
	if err != nil {
		return fmt.Errorf("error writing to lock file %v; %v", lockFile, err)
	} else {
		return nil
	}
}

// GetLock checks existence and contents of lockfile
func GetLock(cfg *utils.Config) (string, error) {
	lockFile := cfg.AidaDb + ".autogen.lock"

	// Read lockfile contents
	content, err := os.ReadFile(lockFile)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("error reading from file; %v", err)
	}

	return string(content), nil
}

// AutogenRun is used to record/update aida-db
func AutogenRun(cfg *utils.Config, g *Generator) error {
	g.Log.Noticef("Starting substate generation %d - %d", g.Opera.FirstEpoch, g.TargetEpoch)

	start := time.Now()
	// stop opera to be able to export events
	errCh := startOperaRecording(g.Cfg, g.TargetEpoch)

	// wait for opera recording response
	err, ok := <-errCh
	if ok && err != nil {
		return err
	}
	g.Log.Noticef("Recording (%v) for epoch range %d - %d finished. It took: %v", g.Cfg.OperaDb, g.Opera.FirstEpoch, g.TargetEpoch, time.Since(start).Round(1*time.Second))
	g.Log.Noticef("Total elapsed time: %v", time.Since(g.start).Round(1*time.Second))

	// reopen aida-db
	g.AidaDb, err = rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot create new db; %v", err)
	}
	substate.SetSubstateDbBackend(g.AidaDb)

	err = g.Opera.getOperaBlockAndEpoch(false)
	if err != nil {
		return err
	}

	return g.Generate()
}

// PrepareAutogen initializes a generator object, opera binary and adjust target range
func PrepareAutogen(ctx *cli.Context, cfg *utils.Config) (*Generator, bool, error) {
	// this explicit overwrite is necessary at first autogen run,
	// in later runs the paths are correctly set in adjustMissingConfigValues
	utils.OverwriteDbPathsByAidaDb(cfg)

	g, err := NewGenerator(ctx, cfg)
	if err != nil {
		return nil, false, err
	}

	err = g.Opera.init()
	if err != nil {
		return nil, false, err
	}

	// user specified targetEpoch
	if cfg.TargetEpoch > 0 {
		g.TargetEpoch = cfg.TargetEpoch
	} else {
		err = g.calculatePatchEnd()
		if err != nil {
			return nil, false, err
		}
	}

	MustCloseDB(g.AidaDb)

	// start epoch is last epoch + 1
	if g.Opera.FirstEpoch > g.TargetEpoch {
		return g, false, nil
	}
	return g, true, nil
}
