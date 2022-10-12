// Package trace provides cli for recording and replaying storage traces.
package trace

import (
	"fmt"
	"strconv"
	cli "gopkg.in/urfave/cli.v1"
)

// chain id is needed for executing EVM in trace record
var chainID int

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

// SetBlockRange checks validity of a block range from command line arguments and
// returns the first and last block as uint 64
func SetBlockRange(firstArg string, lastArg string) (uint64, uint64, error) {
	var err error = nil
	first, ferr := strconv.ParseUint(firstArg, 10, 64)
	last, lerr := strconv.ParseUint(lastArg, 10, 64)
	if ferr != nil || lerr != nil {
		err = fmt.Errorf("substate-cli replay: error in parsing parameters: block number not an integer")
	} else if first < 0 || last < 0 {
		err = fmt.Errorf("substate-cli replay: error: block number must be greater than 0")
	} else if first > last {
		err = fmt.Errorf("substate-cli replay: error: first block has larger number than last block")
	}
	return first, last, err
}
