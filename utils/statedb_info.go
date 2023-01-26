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

	"github.com/ethereum/go-ethereum/common"
)

type StateDbInfo struct {
	Impl     string      `json:"db-impl"`
	Variant  string      `json:"db-variant"`
	Block    uint64      `json:"block"`
	RootHash common.Hash `json:"roothash"`
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

// WriteStateDbInfo write stateDB implementation info and block height to a file
// for a compatibility check when reusing
func WriteStateDbInfo(directory string, cfg *TraceConfig, block uint64, root common.Hash) error {
	dbinfo := &StateDbInfo{Impl: cfg.DbImpl, Variant: cfg.DbVariant, Block: block, RootHash: root}
	filename := filepath.Join(directory, "statedb_info.json")
	jsonByte, err := json.Marshal(dbinfo)
	if err != nil {
		return fmt.Errorf("Failed to encode stateDB info in JSON format")
	}
	if err := os.WriteFile(filename, jsonByte, 0444); err != nil {
		return fmt.Errorf("Failed to write stateDB info to file %v. %v\n", filename, err)
	}
	return nil
}

func ReadStateDbInfo(directory string, cfg *TraceConfig) (StateDbInfo, error) {
	var dbinfo StateDbInfo
	file, err := os.ReadFile(filepath.Join(directory, "statedb_info.json"))
	if err != nil {
		return dbinfo, fmt.Errorf("Failed to read %v in %v. %v", file, cfg.StateDbDir, err)
	}
	if err := json.Unmarshal(file, &dbinfo); err != nil {
		return dbinfo, err
	}
	// check DB type
	if dbinfo.Impl != cfg.DbImpl {
		return dbinfo, fmt.Errorf("Wrong DB implementation.\n\thave %v\n\twant %v", dbinfo.Impl, cfg.DbImpl)
	} else if dbinfo.Variant != cfg.DbVariant {
		return dbinfo, fmt.Errorf("Wrong DB variant.\n\thave %v\n\twant %v", dbinfo.Variant, cfg.DbVariant)
	} else if dbinfo.Block+1 != cfg.First { //the first block must start after the head block in the stateDB
		return dbinfo, fmt.Errorf("The first block is earlier than stateDB.\n\thave %v\n\twant %v", dbinfo.Block+1, cfg.First)
	}
	return dbinfo, nil
}

// RenameTempStateDBDirectory renames a temp directory to a meaningful name if KeepStateDB is set
func RenameTempStateDBDirectory(cfg *TraceConfig, oldDirectory string, block uint64, preloaded bool) {
	newDirectory := oldDirectory
	if cfg.DbImpl != "geth" {
		newDirectory = fmt.Sprintf("state_db_%v_%v_%v", cfg.DbImpl, cfg.DbVariant, block)
	} else {
		newDirectory = fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, block)
	}
	if preloaded {
		newDirectory = filepath.Join(filepath.Join(cfg.StateDbDir, ".."), newDirectory)
	} else {
		newDirectory = filepath.Join(cfg.StateDbDir, newDirectory)
	}
	if err := os.Rename(oldDirectory, newDirectory); err != nil {
		log.Printf("WARNING: failed to rename state directory. %v\n", err)
	}
	log.Printf("StateDB directory: %v\n", newDirectory)
}
