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

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
)

type gethTest interface {
	*stJSON
	setLabel(label string)
}

// NewGethTestDecoder opens all JSON tests within path
func NewGethTestDecoder(cfg *utils.Config) (*GethTestDecoder, error) {
	tests, err := getTestsWithinPath(cfg, utils.StateTests)
	if err != nil {
		return nil, err
	}

	return &GethTestDecoder{
		cfg:   cfg,
		log:   logger.NewLogger(cfg.LogLevel, "state-test-decoder"),
		jsons: tests,
	}, nil
}

type GethTestDecoder struct {
	cfg   *utils.Config
	log   logger.Logger
	jsons []*stJSON
}

// getTestsWithinPath returns all tests in given directory (and subdirectories)
// T is the type into which we want to unmarshal the tests.
func getTestsWithinPath[T gethTest](cfg *utils.Config, testType utils.GethTestType) ([]T, error) {
	path := cfg.ArgPath
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		tests, err := readTestsFromFile[T](path)
		if err != nil {
			return nil, err
		}
		return tests, nil
	}

	switch testType {
	case utils.StateTests:
		gst := path + "/GeneralStateTests"
		_, err := os.Stat(gst)
		if !os.IsNotExist(err) {
			path = gst
		}
	case utils.BlockTests:
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
		toAppend, err := readTestsFromFile[T](p)
		if err != nil {
			return nil, err
		}
		tests = append(tests, toAppend...)
	}

	return tests, err
}

func readTestsFromFile[T gethTest](path string) ([]T, error) {
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

	label := getTestFileLabel(path)

	for _, t := range b {
		t.setLabel(label)
		tests = append(tests, t)
	}
	return tests, nil
}

// getTestFileLabel returns the last folder name and the filename of the given path
func getTestFileLabel(path string) string {
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
