package ethtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	blockchaintest "github.com/Fantom-foundation/Aida/ethtest/blockchaintest"
	statetest "github.com/Fantom-foundation/Aida/ethtest/statetest"
	"github.com/Fantom-foundation/Aida/ethtest/util"
	"github.com/Fantom-foundation/Aida/utils"
)

type ethTest interface {
	*statetest.StJSON | *blockchaintest.BtJSON
	SetLabel(string)
}

// GetTestsWithinPath returns all tests in given directory (and subdirectories)
// T is the type into which we want to unmarshal the tests.
func GetTestsWithinPath[T ethTest](path string, testType string) ([]T, error) {
	switch testType {
	case utils.EthStateTests:
		gst := path + "/GeneralStateTests"
		_, err := os.Stat(gst)
		if !os.IsNotExist(err) {
			path = gst
		}
	case utils.EthBlockChainTests:
		gst := path + "/BlockchainTests"
		_, err := os.Stat(gst)
		if !os.IsNotExist(err) {
			path = gst
		}
	default:
		return nil, errors.New("please choose which testType do you want to read")
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

		t, err := readTestsFromFile[T](p)
		if err != nil {
			return nil, fmt.Errorf("cannot read tests from file %v; %w", p, err)
		}
		tests = append(tests, t...)
	}

	return tests, err
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

// OpenBlockChainTests opens
func OpenBlockChainTests(path string) ([]*blockchaintest.BtJSON, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var allTests []*blockchaintest.BtJSON

	if info.IsDir() {
		allTests, err = GetTestsWithinPath[*blockchaintest.BtJSON](path, utils.EthBlockChainTests)
		if err != nil {
			return nil, err
		}
	} else {
		allTests, err = readTestsFromFile[*blockchaintest.BtJSON](path)
		if err != nil {
			return nil, err
		}
	}

	var dividedTests []*blockchaintest.BtJSON
	for _, t := range allTests {
		for _, n := range util.UsableForks {
			if t.Network == n {
				t.Blocks[0].BlockHeader.Number.SetUint64(utils.KeywordBlocks[250][strings.ToLower(n)] + 1)
				dividedTests = append(dividedTests, t)
				continue
			}
		}
	}

	return dividedTests, nil
}

// OpenStateTests opens
func OpenStateTests(path string) ([]*statetest.StJSON, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var tests []*statetest.StJSON

	if info.IsDir() {
		tests, err = GetTestsWithinPath[*statetest.StJSON](path, utils.EthStateTests)
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
