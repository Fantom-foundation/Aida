package main

import (
	"os"
	"testing"
)

// TestPositiveRecord executes record command for 101 blocks
// Note: Substate.test contains substate from block 5000000 to 5000100
func TestPositiveRecord(t *testing.T) {
	app := initTraceApp()
	os.Args = []string{
		"trace", "record",
		"--trace-file", testTraceDir,
		"--substatedir", "substate.test",
		"5000000", "5000100",
	}
	if err := app.Run(os.Args); err != nil {
		t.Fatalf("%v\n", err)
	}
}

// TestPositiveReplay executes replay command for 101 block
func TestPositiveReplaySubstate(t *testing.T) {
	app := initTraceApp()
	// record
	os.Args = []string{
		"trace", "record",
		"--trace-file", testTraceDir,
		"--substatedir", "substate.test",
		"5000000", "5000100",
	}
	if err := app.Run(os.Args); err != nil {
		t.Fatalf("%v\n", err)
	}
	// replay
	dbTypes := []string{"memory", "geth", "carmen"}
	for _, db := range dbTypes {
		os.Args = []string{
			"trace", "replay-substate",
			"--trace-file", testTraceDir,
			"--db-impl", db,
			"--substatedir", "substate.test",
			"5000000", "5000100",
		}
		if err := app.Run(os.Args); err != nil {
			t.Fatalf("Failed to replay using %v. %v\n", db, err)
		}
	}
}

// TestPositiveReplayValidate executes replay command then validate last state
func TestPositiveReplaySubstateValidate(t *testing.T) {
	app := initTraceApp()
	// record
	os.Args = []string{
		"trace", "record",
		"--trace-file", testTraceDir,
		"--substatedir", "substate.test",
		"5000000", "5000100",
	}
	if err := app.Run(os.Args); err != nil {
		t.Fatalf("%v\n", err)
	}
	// replay
	dbTypes := []string{"memory"}
	for _, db := range dbTypes {
		os.Args = []string{
			"trace", "replay-substate",
			"--trace-file", testTraceDir,
			"--db-impl", db,
			"--substatedir", "substate.test",
			"--validate",
			"5000000", "5000100",
		}
		if err := app.Run(os.Args); err != nil {
			t.Fatalf("Failed to replay using %v. %v\n", db, err)
		}
	}
}

// TestPositiveCompareLog executes compare-log command for 101 blocks
func TestPositiveCompareLog(t *testing.T) {
	app := initTraceApp()
	os.Args = []string{
		"trace", "compare-log",
		"--substatedir", "substate.test",
		"--trace-file", testTraceDir,
		"5000000", "5000100",
	}
	if err := app.Run(os.Args); err != nil {
		t.Fatalf("%v\n", err)
	}
}
