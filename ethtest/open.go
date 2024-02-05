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

var usableForks = []string{"London", "Berlin", "Istanbul", "MuirGlacier"}

type jsonTestType byte

type jsonTest interface {
	*StJSON
}

// GetTestsWithinPath returns all tests in given directory (and subdirectories)
// T is the type into which we want to unmarshal the tests.
func GetTestsWithinPath[T jsonTest](path string, testType jsonTestType) ([]T, error) {
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

	paths, err := utils.GetDirectoryFiles(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read files within directory %v; %v", path, err)
	}

	var tests []T

	for _, p := range paths {
		// todo these directories contain more complex tests, exclude them for now
		if strings.Contains(p, "Shanghai") {
			continue
		}
		if strings.Contains(p, "Cancun") {
			continue
		}
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
