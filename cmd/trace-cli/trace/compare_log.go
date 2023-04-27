package trace

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/urfave/cli/v2"
)

// TraceCompareLogCommand data structure for compare-log command.
var TraceCompareLogCommand = cli.Command{
	Action:    traceCompareLogAction,
	Name:      "compare-log",
	Usage:     "compares storage debug log between record and replay",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.ChainIDFlag,
		&utils.QuietFlag,
		&utils.StateDbImplementationFlag,
		&substate.SubstateDirFlag,
		&substate.WorkersFlag,
		&utils.TraceDebugFlag,
		&utils.TraceFileFlag,
		&utils.AidaDbFlag,
		&utils.LogLevel,
	},
	Description: `
The trace compare-log command requires two arguments:
<blockNumFirst> <blockNumLast>

<blockNumFirst> and <blockNumLast> are the first and
last block of the inclusive range of blocks to replay storage traces.`,
}

// captureDebugLog captures debug log in a string buffer.
func captureDebugLog(traceFunc func(*cli.Context) error, ctx *cli.Context) (string, error) {
	defer func(stdout *os.File) {
		os.Stdout = stdout
	}(os.Stdout)

	// create tmp file storing debug traces
	tmpfile, fileErr := os.CreateTemp("", "debug_trace_tmp")
	if fileErr != nil {
		return "", fileErr
	}
	tmpname := tmpfile.Name()
	// remove tmpfile
	defer os.Remove(tmpname)

	// redirect stdout to tmp file
	os.Stdout = tmpfile

	// run trace record/replay
	err := traceFunc(ctx)

	fileErr = tmpfile.Close()
	if fileErr != nil {
		return "", fileErr
	}
	// copy the output from tmp file
	debugMessage, fileErr := ioutil.ReadFile(tmpname)
	if fileErr != nil {
		return "", fileErr
	}

	return string(debugMessage), err
}

// isLogEqual returns true if input debug traces are identical.
func isLogEqual(record string, replay string) bool {
	// remove log messages from substateDB before comparing
	re := regexp.MustCompile("(?m)[\r\n]+^.*record-replay.*$")
	record = re.ReplaceAllString(record, "")
	replay = re.ReplaceAllString(replay, "")
	return record == replay
}

// traceCompareLogAction implements trace command for validating record and replay debug log.
func traceCompareLogAction(ctx *cli.Context) error {
	log := utils.NewLogger(ctx.String(utils.LogLevel.Name), "Trace Compare Log")

	// process arguments
	if ctx.Args().Len() != 2 {
		return fmt.Errorf("trace compare-log command requires exactly 2 arguments")
	}

	// enable debug-trace
	if !ctx.IsSet(utils.TraceDebugFlag.Name) {
		ctxErr := ctx.Set(utils.TraceDebugFlag.Name, "true")
		if ctxErr != nil {
			return ctxErr
		}
	}
	// disable progress log
	if !ctx.IsSet(utils.QuietFlag.Name) {
		ctxErr := ctx.Set(utils.QuietFlag.Name, "true")
		if ctxErr != nil {
			return ctxErr
		}
	}

	log.Notice("Capture record trace...")
	recordLog, recErr := captureDebugLog(traceRecordAction, ctx)
	if recErr != nil {
		return recErr
	}

	log.Notice("Capture replay trace...")
	replayLog, repErr := captureDebugLog(traceReplaySubstateAction, ctx)
	if repErr != nil {
		return recErr
	}

	log.Notice("Compare traces...")
	if !isLogEqual(recordLog, replayLog) {
		return fmt.Errorf("trace compare-log: Replay trace doesn't match record trace")
	} else {
		log.Notice("Replay trace matches record trace")
	}

	return nil
}
