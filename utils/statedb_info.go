// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const PathToDbInfo = "statedb_info.json"

// StateDbInfo StateDB meta information
type StateDbInfo struct {
	Impl           string      `json:"dbImpl"`         // type of db engine
	Variant        string      `json:"dbVariant"`      // type of db variant
	ArchiveMode    bool        `json:"archiveMode"`    // archive mode
	ArchiveVariant string      `json:"archiveVariant"` // archive variant
	Schema         int         `json:"schema"`         // DB schema version used
	Block          uint64      `json:"block"`          // last block height
	RootHash       common.Hash `json:"rootHash"`       // root hash of the last block height
	GitCommit      string      `json:"gitCommit"`      // Aida git version when creating stateDB
	CreateTime     string      `json:"createTimeUTC"`  // time of creation in utc timezone
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

// CopyDir copies a whole directory recursively
func CopyDir(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}
	if fds, err = ioutil.ReadDir(src); err != nil {
		os.RemoveAll(dst)
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = CopyDir(srcfp, dstfp); err != nil {
				os.RemoveAll(dst)
				return err
			}
		} else {
			if err = copyFile(srcfp, dstfp); err != nil {
				os.Remove(dst)
				return err
			}
		}
	}
	return nil
}

// WriteStateDbInfo writes stateDB implementation info and block height to a file
// for a compatibility check when reloading
func WriteStateDbInfo(directory string, cfg *Config, block uint64, root common.Hash) error {
	dbinfo := &StateDbInfo{
		Impl:           cfg.DbImpl,
		Variant:        cfg.DbVariant,
		ArchiveMode:    cfg.ArchiveMode,
		ArchiveVariant: cfg.ArchiveVariant,
		Schema:         cfg.CarmenSchema,
		Block:          block,
		RootHash:       root,
		GitCommit:      GitCommit,
		CreateTime:     time.Now().UTC().Format(time.UnixDate),
	}
	filename := filepath.Join(directory, PathToDbInfo)
	jsonByte, err := json.MarshalIndent(dbinfo, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to encode stateDB info in JSON format")
	}
	if err := os.WriteFile(filename, jsonByte, 0666); err != nil {
		return fmt.Errorf("Failed to write stateDB info to file %v. %v\n", filename, err)
	}
	return nil
}

// ReadStateDbInfo reads meta file of loaded stateDB
func ReadStateDbInfo(filename string) (StateDbInfo, error) {
	var dbinfo StateDbInfo
	file, err := os.ReadFile(filename)
	if err != nil {
		return dbinfo, fmt.Errorf("failed to read %v; %v", filename, err)
	}
	err = json.Unmarshal(file, &dbinfo)
	return dbinfo, err
}

// RenameTempStateDbDirectory renames a temp directory to a meaningful name
func RenameTempStateDbDirectory(cfg *Config, oldDirectory string, block uint64) string {
	var newDirectory string
	// if custom db name is given, use it. Otherwise, generate readable name from db info.
	if cfg.CustomDbName != "" {
		newDirectory = cfg.CustomDbName
	} else if cfg.DbImpl != "geth" {
		newDirectory = fmt.Sprintf("state_db_%v_%v_%v", cfg.DbImpl, cfg.DbVariant, block)
	} else {
		newDirectory = fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, block)
	}
	newDirectory = filepath.Join(cfg.DbTmp, newDirectory)
	if err := os.Rename(oldDirectory, newDirectory); err != nil {
		log.Printf("WARNING: failed to rename state directory. %v\n", err)
		newDirectory = oldDirectory
	}
	return newDirectory
}
