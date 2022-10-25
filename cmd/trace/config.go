// Package trace provides cli for recording and replaying storage traces.
package trace

import (
	"fmt"
	"strconv"

	"github.com/urfave/cli/v2"
)

// Chain id for recording either mainnet or testnets.
var chainID int

// Trace debugging flag
var traceDebug bool = false

// Command line options for common flags in record and replay
var (
	chainIDFlag = cli.IntFlag{
		Name:  "chainid",
		Usage: "ChainID for replayer",
		Value: 250,
	}
	cpuProfileFlag = cli.StringFlag{
		Name:  "cpuprofile",
		Usage: "enables CPU profiling",
	}
	profileFlag = cli.BoolFlag{
		Name:  "profile",
		Usage: "enables profiling",
	}
	disableProgressFlag = cli.BoolFlag{
		Name:  "disable-progress",
		Usage: "disable progress report",
	}
	stateDbImplementation = cli.StringFlag{
		Name:  "db-impl",
		Usage: "select state DB implementation",
		Value: "memory",
	}
	stateDbVariant = cli.StringFlag{
		Name:  "db-variant",
		Usage: "select a state DB variant",
		Value: "",
	}
	traceDebugFlag = cli.BoolFlag{
		Name:  "trace-debug",
		Usage: "enable debug output for tracing",
	}
	traceDirectoryFlag = cli.StringFlag{
		Name:  "trace-dir",
		Usage: "set storage trace's output directory",
		Value: "./",
	}
	validateEndState = cli.BoolFlag{
		Name:  "validate",
		Usage: "enables end-state validation",
	}
)

// Check the validity of a block range and return the first and last block as numbers.
func SetBlockRange(firstArg string, lastArg string) (uint64, uint64, error) {
	var err error = nil
	first, ferr := strconv.ParseUint(firstArg, 10, 64)
	last, lerr := strconv.ParseUint(lastArg, 10, 64)
	if ferr != nil || lerr != nil {
		err = fmt.Errorf("error: block number not an integer")
	} else if first < 0 || last < 0 {
		err = fmt.Errorf("error: block number must be greater than 0")
	} else if first > last {
		err = fmt.Errorf("error: first block has larger number than last block")
	}
	return first, last, err
}
