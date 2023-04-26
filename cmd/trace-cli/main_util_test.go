package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	testTraceFile = "trace-test/trace.dat"
	testTraceDir = "trace-test"
)

// TestMain runs global setup, test cases then global teardown
func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

// setup prepares substateDB and creates trace directory
func setup() {
	// download and extract substate.test
	err := setupTestSubstateDB()
	if err != nil {
		fmt.Errorf("unable to load substatedb. %v", err)
	}

	// create trace directory
	err = os.Mkdir(testTraceDir, 0700)
	if err != nil {
		fmt.Errorf("unable to create direcotry. %v", err)
	}

	fmt.Printf("Setup completed\n")
}

// teardown removes temp directories
func teardown() {
	// Do something here.
	os.RemoveAll(testTraceDir)
	os.RemoveAll("substate.test")
	fmt.Printf("Teardown completed\n")
}

// setupTestSubstateDB downloads compressed substates and extract in local directory
func setupTestSubstateDB() error {
	// download substate.test from url
	// set timeout to 1 minutes
	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get("https://github.com/Fantom-foundation/substate-cli/releases/download/substate-test/substate.test.tar.gz")
	if err != nil {
		return err
	}

	// channel downloaded content to gzip stream
	gzipStream, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzipStream.Close()

	tarReader := tar.NewReader(gzipStream)

	// decompress and store each file in archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// if head is a directory, create a new directory
		if header.Typeflag == tar.TypeDir {
			if err = os.MkdirAll(header.Name, 0700); err != nil {
				return err
			}
			// if not a directory, copy to out file
		} else {
			outFile, err := os.OpenFile(header.Name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
			if err != nil {
				return err
			}
			defer outFile.Close()
			if _, err = io.Copy(outFile, tarReader); err != nil {
				return err
			}
		}
	}
	return nil
}
