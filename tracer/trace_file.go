package tracer

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Fantom-foundation/Aida/utils"
	"github.com/dsnet/compress/bzip2"
)

const ReaderBufferSize = 65536 * 256 // 16MiB

// TraceFile data structure for reading a trace file
type TraceFile struct {
	firstBlock uint64        // first block in trace file
	file       *os.File      // trace file
	reader     *bufio.Reader // read buffer
	zreader    *bzip2.Reader // compressed stream
}

// NewTraceFile opens a file, read header and create a TraceFile object.
func NewTraceFile(fname string) (*TraceFile, error) {
	tf := new(TraceFile)

	// open a bzip file
	var err error
	tf.file, err = os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("cannot open trace file; %v", err)
	}
	tf.zreader, err = bzip2.NewReader(tf.file, &bzip2.ReaderConfig{})
	if err != nil {
		return nil, fmt.Errorf("cannot open bzip stream; %v", err)
	}
	tf.reader = bufio.NewReaderSize(tf.zreader, ReaderBufferSize)

	//read first block
	var header [8]byte
	if _, err := io.ReadFull(tf.reader, header[:]); err != nil {
		return nil, fmt.Errorf("fail to read file  header; %v", err)
	}
	tf.firstBlock = binary.LittleEndian.Uint64(header[:])
	return tf, nil
}

// Release closes all file channels.
func (tf *TraceFile) Release() error {
	if err := tf.zreader.Close(); err != nil {
		return fmt.Errorf("cannot close compressed stream. %v", err)
	}
	if err := tf.file.Close(); err != nil {
		return fmt.Errorf("cannot close trace file. %v", err)
	}
	return nil
}

// keepRelevantTraceFiles remove out trace files whose first block is
// out of range of the specified range.
// 1. a trace file contains blocks larger than the specified range
// 2. a trace file contains blocks prior to the specified range.
func keepRelevantTraceFiles(first, last uint64, sortedList []uint64, blockFile map[uint64]string) []string {
	var (
		highestFirstBlock uint64
		traceFiles        []string
	)
	for _, fileFirstBlock := range sortedList {
		// remove a file if the first block is larger than the target last block
		if fileFirstBlock > last {
			delete(blockFile, fileFirstBlock)
			continue
		}
		if fileFirstBlock > highestFirstBlock && fileFirstBlock <= first {
			// delete a file with a range lower than the first block
			if _, found := blockFile[highestFirstBlock]; found {
				delete(blockFile, highestFirstBlock)
			}
			highestFirstBlock = fileFirstBlock
		}
	}

	// add relevant trace files to the return list
	for _, fileFirstBlock := range sortedList {
		if _, found := blockFile[fileFirstBlock]; found {
			traceFiles = append(traceFiles, blockFile[fileFirstBlock])
		}
	}
	return traceFiles
}

// GetTraceFiles returns a list of valid trace files for the specified range.
// The files is sorted by ascending order of first block.
func GetTraceFiles(cfg *utils.Config) ([]string, error) {
	var traceFiles []string
	// load trace files from a directory if given
	if cfg.TraceDirectory != "" {
		var firstBlockList []uint64
		blockFile := make(map[uint64]string)

		// read files in a directory
		dir, err := os.Open(cfg.TraceDirectory)
		if err != nil {
			return traceFiles, err
		}
		defer dir.Close()
		fileList, err := dir.Readdirnames(0)
		if err != nil {
			return traceFiles, err
		}

		// for each trace file get its first block
		for _, name := range fileList {
			fname := filepath.Join(cfg.TraceDirectory, name)
			tf, err := NewTraceFile(fname)
			if err != nil {
				return traceFiles, err
			}
			first := tf.firstBlock
			if err := tf.Release(); err != nil {
				return traceFiles, err
			}
			blockFile[first] = fname
			firstBlockList = append(firstBlockList, first)
		}

		// sort first block
		sort.Slice(firstBlockList, func(i, j int) bool { return firstBlockList[i] < firstBlockList[j] })
		// remove any if trace files which are not in range the target range
		traceFiles = keepRelevantTraceFiles(cfg.First, cfg.Last, firstBlockList, blockFile)

	} else if cfg.TraceFile != "" {
		tf, err := NewTraceFile(cfg.TraceFile)
		if err != nil {
			return traceFiles, err
		}
		first := tf.firstBlock
		// exclude the file if it starts after the last target block
		if first <= cfg.Last {
			traceFiles = append(traceFiles, cfg.TraceFile)
		}
	} else {
		return traceFiles, fmt.Errorf("trace file is not found.")
	}
	if len(traceFiles) == 0 {
		return traceFiles,
			fmt.Errorf("no trace files for the specified block range %v - %v", cfg.First, cfg.Last)
	}
	return traceFiles, nil
}
