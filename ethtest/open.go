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
	"strings"

	"github.com/Fantom-foundation/Aida/utils"
)

const (
	BlockTests jsonTestType = iota
	StateTests
)

var usableForks = []string{"London", "Berlin", "Istanbul", "MuirGlacier", "TestNetwork"}

type jsonTestType byte

type stateTest interface {
	*StJSON
}

// GetTestsWithinPath returns all tests in given directory (and subdirectories)
// T is the type into which we want to unmarshal the tests.
func GetTestsWithinPath[T stateTest](path string, testType jsonTestType) ([]T, error) {
	switch testType {
	case StateTests:
		gst := path + "/GeneralStateTests"
		_, err := os.Stat(gst)
		if !os.IsNotExist(err) {
			path = gst
		}
	case BlockTests:
		return nil, errors.New("block testType not yet implemented")
	default:
		return nil, errors.New("please chose which testType do you want to read")
	}

	paths, err := utils.GetDirectoryFiles(".json", path)
	if err != nil {
		return nil, fmt.Errorf("cannot read files within directory %v; %v", path, err)
	}

	var tests []T

	for _, p := range paths {
		// todo these directories contain more complex tests, exclude them for now
		if strings.Contains(p, "VMTests") {
			continue
		}
		if strings.Contains(p, "stArgsZeroOneBalance") {
			continue
		}

		// TODO merge usability with readTestsFromFile
		file, err := os.Open(p)
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
			//return nil, fmt.Errorf("cannot unmarshal file %v", p)
			fmt.Printf("SKIPPED: cannot unmarshal file %v\n", p)
			continue
		}

		testLabel := getTestLabel(p)

		for _, t := range b {
			(*t).TestLabel = testLabel
			tests = append(tests, t)
		}
	}

	return tests, err
}

// OpenStateTests opens
func OpenStateTests(path string) ([]*StJSON, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var tests []*StJSON

	if info.IsDir() {
		tests, err = GetTestsWithinPath[*StJSON](path, StateTests)
		if err != nil {
			return nil, err
		}

	} else {
		tests, err = readTestsFromFile(path)
		if err != nil {
			return nil, err
		}
	}

	return tests, nil
}

func readTestsFromFile(path string) ([]*StJSON, error) {
	var tests []*StJSON
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	byteJSON, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var b map[string]*StJSON
	err = json.Unmarshal(byteJSON, &b)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal file %v", path)
	}

	testLabel := getTestLabel(path)

	for _, t := range b {
		t.TestLabel = testLabel
		tests = append(tests, t)
	}
	return tests, nil
}

// getTestLabel returns the last folder name and the filename of the given path
func getTestLabel(path string) string {
	// Split the path into components
	pathComponents := strings.Split(path, "/")

	var lastFolderName = ""
	var filename = ""

	if len(pathComponents) > 1 {
		// Extract the last folder name
		lastFolderName = pathComponents[len(pathComponents)-2]
	}

	if len(pathComponents) > 0 {
		// Extract the filename
		filename = pathComponents[len(pathComponents)-1]
	}
	return lastFolderName + "/" + filename
}
