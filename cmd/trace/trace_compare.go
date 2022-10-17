package trace

import (
	"fmt"
	"github.com/ethereum/go-ethereum/substate"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"regexp"
)

// Trace debug compare command
var TraceDebugCompareCommand = cli.Command{
	Action:    traceDebugCompareAction,
	Name:      "debug-compare",
	Usage:     "compares storage traces from record and replay",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&chainIDFlag,
		&stateDbImplementation,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&traceDebugFlag,
		&traceDirectoryFlag,
	},
	Description: `
The trace replay command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// captureDebugTrace captures debug messages and stores to string buffer.
func captureDebugTrace(traceFunc func(*cli.Context) error, ctx *cli.Context) (string, error) {
	defer func(stdout *os.File) {
		os.Stdout = stdout
	}(os.Stdout)

	tmpfile, fileErr := os.CreateTemp("", "debug_trace_tmp")
	if fileErr != nil {
		return "", fileErr
	}
	tmpname := tmpfile.Name()
	defer os.Remove(tmpname)

	// redirect stdout to a temp file
	os.Stdout = tmpfile

	// run trace record/replay
	err := traceFunc(ctx)

	fileErr = tmpfile.Close()
	if fileErr != nil {
		return "", fileErr
	}
	// copy the output from temp file
	debugMessage, fileErr := ioutil.ReadFile(tmpname)
	if fileErr != nil {
		return "", fileErr
	}

	return string(debugMessage), err
}

// isTraceEqual returns true if input debug traces are identical.
func isTraceEqual(record string, replay string) bool {
	re := regexp.MustCompile("(?m)[\r\n]+^.*record-replay.*$")
	record = re.ReplaceAllString(record, "")
	replay = re.ReplaceAllString(replay, "")
	return record == replay
}

// traceDebugCompareAction mplements trace command for validating record and replay.
func traceDebugCompareAction(ctx *cli.Context) error {
	// process arguments
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace replay-trace command requires exactly 2 arguments")
	}
	if !ctx.IsSet(traceDebugFlag.Name) {
		ctxErr := ctx.Set(traceDebugFlag.Name, "true")
		if ctxErr != nil {
			return ctxErr
		}
	}
	fmt.Printf("Capture record trace\n")
	recordTrace, recErr := captureDebugTrace(traceRecordAction, ctx)
	if recErr != nil {
		return recErr
	}
	fmt.Printf("Capture replay trace\n")
	replayTrace, repErr := captureDebugTrace(traceReplayAction, ctx)
	if repErr != nil {
		return recErr
	}

	if !isTraceEqual(recordTrace, replayTrace) {
		return fmt.Errorf("Replay trace doesn't match record trace.")
	} else {
		fmt.Printf("Replay trace matches record trace.\n")
	}

	return nil
}
