package trace

import (
	"fmt"
	cli "gopkg.in/urfave/cli.v1"
	"strconv"
)

// chain id
var chainID int
var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate   = ""
)

// trace debugging
var traceDebug bool = false

// command line options
var (
	ChainIDFlag = cli.IntFlag{
		Name:  "chainid",
		Usage: "ChainID for replayer",
		Value: 250,
	}
	TraceDirectoryFlag = cli.StringFlag{
		Name:  "trace-dir",
		Usage: "set storage trace's output directory",
		Value: "./",
	}
	TraceDebugFlag = cli.BoolFlag{
		Name:  "trace-debug",
		Usage: "enable debug output for tracing",
	}
)

func SetBlockRange(firstArg string, lastArg string) (uint64, uint64, error) {
	first, ferr := strconv.ParseUint(firstArg, 10, 64)
	last, lerr := strconv.ParseUint(lastArg, 10, 64)
	if ferr != nil || lerr != nil {
		return first, last, fmt.Errorf("substate-cli replay: error in parsing parameters: block number not an integer")
	}
	if first < 0 || last < 0 {
		return first, last, fmt.Errorf("substate-cli replay: error: block number must be greater than 0")
	}
	if first > last {
		return first, last, fmt.Errorf("substate-cli replay: error: first block has larger number than last block")
	}
	return first, last, nil
}
