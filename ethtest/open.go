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

package ethtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/utils"
)

type ethTest interface {
	*stJSON
	setPath(path string)
	setDescription(desc string)
	getGeneratedTestHash() string
}

// getTestsWithinPath returns all tests in given directory (and subdirectories)
// T is the type into which we want to unmarshal the tests.
func getTestsWithinPath[T ethTest](cfg *utils.Config, testType utils.EthTestType) ([]T, error) {
	path := cfg.ArgPath
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Check for single file
	if !info.IsDir() {
		tests, err := readTestsFromFile[T](path, cfg.EthTestHash)
		if err != nil {
			return nil, err
		}
		return tests, nil
	}

	// Get all files for given test types
	var dirPaths []string
	switch testType {
	case utils.StateTests:
		// If all dir with all tests is passed, only runnable StateTests are extracted
		gst := path + "/GeneralStateTests"
		_, err = os.Stat(gst)
		if err == nil {
			dirPaths = append(dirPaths, gst)
		}

		eipt := filepath.Join(path, "EIPTests/StateTests")
		_, err = os.Stat(eipt)
		if err == nil {
			dirPaths = append(dirPaths, eipt)
		}

		// Otherwise exact directory with tests is passed
		if len(dirPaths) == 0 {
			dirPaths = []string{path}
		}
	case utils.BlockTests:
		return nil, errors.New("blockchain test-type not yet implemented")
	default:
		return nil, errors.New("please chose which testType do you want to read")
	}

	filePaths, err := utils.GetFilesWithinDirectories(".json", []string{path})
	if err != nil {
		return nil, fmt.Errorf("cannot read files within directory %v; %v", path, err)
	}

	var tests []T

	for _, p := range filePaths {
		toAppend, err := readTestsFromFile[T](p, cfg.EthTestHash)
		if err != nil {
			return nil, err
		}
		tests = append(tests, toAppend...)
	}

	return tests, err
}

func readTestsFromFile[T ethTest](path string, testHash string) ([]T, error) {
	var tests []T
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	byteJSON, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var b map[string]T
	err = json.Unmarshal(byteJSON, &b)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal file %v", path)
	}

	for desc, t := range b {
		t.setPath(path)
		t.setDescription(desc)
		// do we want to run a single test?
		if testHash != "" && testHash == t.getGeneratedTestHash() {
			tests = append(tests, t)
			return tests, nil
		}
		tests = append(tests, t)
	}
	if tests == nil {
		return nil, fmt.Errorf("no tests found for given setup")
	}
	return tests, nil
}
