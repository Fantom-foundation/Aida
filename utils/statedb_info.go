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

const DbInfoName = "statedb_info.json"

// StateDB meta information
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

// copyDir copies a whole directory recursively
func copyDir(src string, dst string) error {
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
			if err = copyDir(srcfp, dstfp); err != nil {
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
	filename := filepath.Join(directory, DbInfoName)
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
		return dbinfo, fmt.Errorf("Failed to read %v. %v", filename, err)
	}
	err = json.Unmarshal(file, &dbinfo)
	return dbinfo, err
}

// RenameTempStateDBDirectory renames a temp directory to a meaningful name
func RenameTempStateDBDirectory(cfg *Config, oldDirectory string, block uint64) {
	var newDirectory string
	if cfg.DbImpl != "geth" {
		newDirectory = fmt.Sprintf("state_db_%v_%v_%v", cfg.DbImpl, cfg.DbVariant, block)
	} else {
		newDirectory = fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, block)
	}
	newDirectory = filepath.Join(cfg.StateDbTempDir, newDirectory)
	if err := os.Rename(oldDirectory, newDirectory); err != nil {
		log.Printf("WARNING: failed to rename state directory. %v\n", err)
		newDirectory = oldDirectory
	}
	log.Printf("StateDB directory: %v\n", newDirectory)
}
