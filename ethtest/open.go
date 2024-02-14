package ethtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	blocktest "github.com/Fantom-foundation/Aida/ethtest/block_test"
	"github.com/Fantom-foundation/Aida/ethtest/state_test"
	"github.com/Fantom-foundation/Aida/utils"
)

const (
	BlockTests jsonTestType = iota
	StateTests
)

type jsonTestType byte

type ethTest interface {
	*statetest.StJSON | *blocktest.BtJSON
	SetLabel(string)
}

// GetTestsWithinPath returns all tests in given directory (and subdirectories)
// T is the type into which we want to unmarshal the tests.
func GetTestsWithinPath[T ethTest](path string, testType jsonTestType) ([]T, error) {
	switch testType {
	case StateTests:
		gst := path + "/GeneralStateTests"
		_, err := os.Stat(gst)
		if !os.IsNotExist(err) {
			path = gst
		}
	case BlockTests:
		gst := path + "/BlockchainTests"
		_, err := os.Stat(gst)
		if !os.IsNotExist(err) {
			path = gst
		}
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
			t.SetLabel(testLabel)
			tests = append(tests, t)
		}
	}

	return tests, err
}

// OpenBlockTests opens
func OpenBlockTests(path string) ([]*blocktest.BtJSON, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var tests []*blocktest.BtJSON

	if info.IsDir() {
		tests, err = GetTestsWithinPath[*blocktest.BtJSON](path, StateTests)
		if err != nil {
			return nil, err
		}
	} else {
		tests, err = readTestsFromFile[*blocktest.BtJSON](path)
		if err != nil {
			return nil, err
		}
	}

	return tests, nil
}

// OpenStateTests opens
func OpenStateTests(path string) ([]*statetest.StJSON, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var tests []*statetest.StJSON

	if info.IsDir() {
		tests, err = GetTestsWithinPath[*statetest.StJSON](path, StateTests)
		if err != nil {
			return nil, err
		}

	} else {
		tests, err = readTestsFromFile[*statetest.StJSON](path)
		if err != nil {
			return nil, err
		}
	}

	return tests, nil
}

func readTestsFromFile[T ethTest](path string) ([]T, error) {
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
		return nil, fmt.Errorf("cannot unmarshal file %v; %v", path, err)
	}

	testLabel := getTestLabel(path)

	for _, t := range b {
		t.SetLabel(testLabel)
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
