package tracer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/dsnet/compress/bzip2"
)

const firstBlockInTrace = 10

// makeTestTraceFileConfig creates a config struct for reading trace files
func makeTestTraceFileConfig() *utils.Config {
	cfg := &utils.Config{
		TraceFile:      "test_trace.dat",
		TraceDirectory: "test_traces",
	}
	return cfg
}

// prepareTraceFile creates a file with file header only
func prepareTraceFile(fname string) error {
	rCtx, err := context.NewRecord(fname, firstBlockInTrace)
	if err != nil {
		return err
	}
	rCtx.Close()
	return nil
}

// prepareTraceDirectory creates a directory and empty trace files with headers
// ranging from 0 to 10000.
func prepareTraceDirectory(fdir string, numFiles int) error {
	// create directory
	if err := os.Mkdir(fdir, 0755); err != nil {
		return err
	}
	startBlock := uint64(0)
	// generate files in the directory
	filePrefix := filepath.Join(fdir, "test_trace_file_")
	for i := 0; i < numFiles; i++ {
		fname := fmt.Sprintf("%v%v.dat", filePrefix, i)
		rCtx, err := context.NewRecord(fname, uint64(startBlock))
		if err != nil {
			return err
		}
		rCtx.Close()
		startBlock += 1000
	}
	return nil
}

// prepareTraceFileWithoutHeader create a special trace file without header.
// This file is used to test error handling.
func prepareTraceFileWithoutHeader(filename string) error {
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file %v already exists", filename)
	}
	// open trace file, write buffer, and compressed stream
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("cannot open trace file; %v", err)
	}
	bFile := bufio.NewWriterSize(file, context.WriteBufferSize)
	zFile, err := bzip2.NewWriter(bFile, &bzip2.WriterConfig{Level: 9})
	if err != nil {
		return fmt.Errorf("cannot open bzip2 stream; %v", err)
	}

	// close file
	if err := zFile.Close(); err != nil {
		return fmt.Errorf("cannot close bzip2 writer; %v", err)
	}
	if err := bFile.Flush(); err != nil {
		return fmt.Errorf("cannot flush buffer; %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("cannot close trace file; %v", err)
	}
	return nil
}

// createBlockFileMap generates a map of block (key) and filename (value)
func createBlockFileMap(numFiles int) (map[uint64]string, []uint64) {
	// create block-filename map
	var (
		filePrefix = "file_" // file prefix string
		blockFile  = make(map[uint64]string)
		first      = uint64(0)
		interval   = uint64(1_000)
		sortedList []uint64
	)
	for i := 0; i < numFiles; i++ {
		blockFile[first] = fmt.Sprintf("%v%v.dat", filePrefix, i)
		sortedList = append(sortedList, first)
		first += interval
	}
	return blockFile, sortedList
}

// Test NewTraceFile and Release functionalities.
func TestTraceFile_NewAndRelease(t *testing.T) {
	// first and last block don't matter in this test
	cfg := makeTestTraceFileConfig()
	if err := prepareTraceFile(cfg.TraceFile); err != nil {
		t.Fatalf("Fail to create a trace file %v; %v", cfg.TraceFile, err)
	}
	defer os.Remove(cfg.TraceFile)
	// open an existing trace file -- expecting no errors
	tf, err := NewTraceFile(cfg.TraceFile)
	if err != nil {
		t.Fatalf("Fail to read a trace file %v; %v", cfg.TraceFile, err)
	}
	if err := tf.Release(); err != nil {
		t.Fatalf("Fail to release a trace file; %v", err)
	}
	// open with wrong file name
	if _, err := NewTraceFile("wrong_file.dat"); err == nil {
		t.Fatalf("Expect file not found error")
	}
	// open a trace file with no first block in the header
	fname := "no_header.dat"
	if err = prepareTraceFileWithoutHeader(fname); err != nil {
		t.Fatalf("Fail to create a trace file %v; %v", cfg.TraceFile, err)
	}
	defer os.Remove(fname)
	if _, err := NewTraceFile(fname); err == nil {
		t.Fatalf("Expect an error reading trace's first block")
	}
}

// Test function keepRelevantTraceFiles. The test ensures that any files not in
// the target range are removed from the return list.
func TestTraceFile_keepRelevantTraceFiles(t *testing.T) {
	var (
		numFiles   = 10
		traceFiles []string
	)

	// no delete
	blockFile, sortedList := createBlockFileMap(numFiles)
	traceFiles = keepRelevantTraceFiles(50, 10010, sortedList, blockFile)
	if len(traceFiles) != numFiles {
		t.Fatalf("Mismatched number of trace files; have %v, want %v", len(blockFile), numFiles)
	}

	// delete the first two files 0-999 and 1000-1999
	blockFile, sortedList = createBlockFileMap(numFiles)
	traceFiles = keepRelevantTraceFiles(2000, 10010, sortedList, blockFile)
	if len(traceFiles) != numFiles-2 {
		t.Fatalf("Mismatched number of trace files; have %v, want %v", len(blockFile), numFiles)
	}

	// delete the last file with range 9000 - 9999
	blockFile, sortedList = createBlockFileMap(numFiles)
	traceFiles = keepRelevantTraceFiles(50, 8999, sortedList, blockFile)
	if len(traceFiles) != numFiles-1 {
		t.Fatalf("Mismatched number of trace files; have %v, want %v", len(blockFile), numFiles)
	}
}

// Test function GetTraceFiles which should return a list of relevant trace file name
// for the specified range.
func TestTraceFile_GetTraceFiles(t *testing.T) {
	// get a trace file from --trace-file
	cfg := &utils.Config{
		TraceFile: "test_trace.dat",
		First:     1500,
		Last:      5500,
	}
	if err := prepareTraceFile(cfg.TraceFile); err != nil {
		t.Fatalf("Fail to create a trace file %v; %v", cfg.TraceFile, err)
	}
	defer os.Remove(cfg.TraceFile)
	list, err := GetTraceFiles(cfg)
	if err != nil {
		t.Fatalf("Fail to retrieve a trace file %v; %v", cfg.TraceFile, err)
	}
	if len(list) != 1 || list[0] != cfg.TraceFile {
		t.Fatalf("No trace files found")
	}

	// get trace files from --trace-dir
	numFiles := 10
	cfg = &utils.Config{
		TraceDirectory: "test_traces",
		First:          1500,
		Last:           5500,
	}
	if err := prepareTraceDirectory(cfg.TraceDirectory, numFiles); err != nil {
		t.Fatalf("Fail to prepare a trace directory %v; %v", cfg.TraceDirectory, err)
	}
	defer os.RemoveAll(cfg.TraceDirectory)
	list, err = GetTraceFiles(cfg)
	if err != nil {
		t.Fatalf("Fail to retrieve a list of trace files from %v; %v", cfg.TraceDirectory, err)
	}
	// expect 5 trace files for range 1500 - 5500
	if len(list) != 5 {
		t.Fatalf("No trace files found")
	}

	// get trace files returns an error if trace files are not given
	cfg = &utils.Config{
		First: 1500,
		Last:  5500,
	}
	list, err = GetTraceFiles(cfg)
	if err == nil {
		t.Fatalf("Trace are not given; expect an error")
	}

}
