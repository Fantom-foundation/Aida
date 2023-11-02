package db

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/op/go-logging"
)

// openSourceDatabases opens all databases required for merge
func openSourceDatabases(sourceDbPaths []string) ([]ethdb.Database, error) {
	if len(sourceDbPaths) < 1 {
		return nil, fmt.Errorf("no source database were specified\n")
	}

	var sourceDbs []ethdb.Database
	for i := 0; i < len(sourceDbPaths); i++ {
		path := sourceDbPaths[i]
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source database %s; doesn't exist\n", path)
		}
		db, err := rawdb.NewLevelDBDatabase(path, 1024, 100, "", true)
		if err != nil {
			return nil, fmt.Errorf("source database %s; error: %v", path, err)
		}
		sourceDbs = append(sourceDbs, db)
	}

	return sourceDbs, nil
}

// MustCloseDB close database safely
func MustCloseDB(db ethdb.Database) {
	if db != nil {
		err := db.Close()
		if err != nil {
			if err.Error() != "leveldb: closed" {
				fmt.Printf("could not close database; %s\n", err.Error())
			}
		}
	}
}

// runCommand wraps cmd execution to distinguish whether to display its output
func runCommand(cmd *exec.Cmd, resultChan chan string, stopChan chan struct{}, log *logging.Logger) error {
	if resultChan != nil {
		defer close(resultChan)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("unable to create StdoutPipe; %v", err)
	}
	defer stdout.Close()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("unable to create StderrPipe; %v", err)
	}
	defer stderr.Close()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("unable to start Command %v; %v", cmd, err)
	}

	merged := io.MultiReader(stderr, stdout)
	scanner := bufio.NewScanner(merged)

	lastOutputMessagesChan := make(chan string, commandOutputLimit)

	// scannedChan to relay command output into channel to be able to select with stopChan
	scannedChan := make(chan string)
	go func() {
		for scanner.Scan() {
			scannedChan <- scanner.Text()
		}
		close(scannedChan)
	}()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// this command expects possibility to be stopped by kill signal from aida
	for {
		select {
		case <-stopChan:
			// not returning any error other than from failure of kill signal,
			// because the command was terminated by aida intentionally
			return killCommand(cmd, log, done)
		case m, ok := <-scannedChan:
			if !ok {
				// set scannedChan to nil to prevent closing it twice - in next for loop cycle scannedChan will be ignored
				scannedChan = nil
				close(lastOutputMessagesChan)
				break
			}
			processScannedCommandOutput(m, resultChan, log, lastOutputMessagesChan)
		case res, ok := <-done:
			return processCommandResult(res, ok, scanner, lastOutputMessagesChan, resultChan, cmd)
		}
	}
}

// killCommand terminates command gracefully first and then forcefully
func killCommand(cmd *exec.Cmd, log *logging.Logger, done chan error) error {
	// A stop signal was received; terminate the command.
	// Attempting to interrupt command gracefully first.
	// Create a timeout with a 1-minute duration.
	timeout := time.NewTimer(time.Minute)
	err := cmd.Process.Signal(syscall.SIGINT)
	if err != nil {
		// might be just race condition when process already finished
		log.Warningf("unable to send SIGINT to Command %v; %v", cmd, err)
	}

	select {
	case <-done:
		log.Noticef("Command %v terminated gracefully", cmd)
	case <-timeout.C:
		// Send a kill signal to the process
		err = cmd.Process.Signal(syscall.SIGKILL)
		if err != nil {
			return fmt.Errorf("unable to send SIGKILL to Command %v; %v", cmd, err)
		}
		// Wait for cmd.Wait() to return after termination.
		<-done
	}
	return nil
}

// processScannedCommandOutput output and send it to resultChan if it is listening and keep lastOutputMessagesChan updated
func processScannedCommandOutput(message string, resultChan chan string, log *logging.Logger, lastOutputMessagesChan chan string) {
	if resultChan != nil {
		resultChan <- message
	}
	if log.IsEnabledFor(logging.DEBUG) {
		log.Debug(message)
	} else {
		// in case debugging is turned off and resultChan doesn't listen to output
		// we need to keep most recent output lines in case of error
		if resultChan == nil {
			// throw out the oldest line in case we are at limit
			if len(lastOutputMessagesChan) == commandOutputLimit {
				<-lastOutputMessagesChan
			}
			lastOutputMessagesChan <- message
		}
	}
}

// processCommandResult is used to process command result
func processCommandResult(err error, ok bool, scanner *bufio.Scanner, lastOutputMessagesChan chan string, resultChan chan string, cmd *exec.Cmd) error {
	if !ok {
		return fmt.Errorf("unexpected doneChan closed error while executing Command %v; %v", cmd, err)
	}
	// command failed
	if err != nil {
		// print out gathered output since generation failed
		for {
			m, ok := <-lastOutputMessagesChan
			if !ok {
				break
			}
			log.Error(m)
		}

		// read rest of the output - might not be needed
		for scanner.Scan() {
			m := scanner.Text()
			if resultChan != nil {
				resultChan <- m
			}
			log.Error(m)
		}
		return fmt.Errorf("error while executing Command %v; %v", cmd, err)
	}
	return nil
}

// calculateMD5Sum calculates MD5 hash of given file
func calculateMD5Sum(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("unable open file %s; %v", filePath, err.Error())
	}
	defer file.Close()

	// Create a new MD5 hash instance
	hash := md5.New()

	// Copy the file content into the hash instance
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", fmt.Errorf("unable to calculate md5; %v", err)
	}

	// Calculate the MD5 checksum as a byte slice
	checksum := hash.Sum(nil)

	// Convert the checksum to a hexadecimal string
	md5sum := hex.EncodeToString(checksum)

	return md5sum, nil
}

// startOperaIpc starts opera node for ipc requests
func startOperaIpc(cfg *utils.Config, stopChan chan struct{}) chan error {
	errChan := make(chan error, 1)

	log := logger.NewLogger(cfg.LogLevel, "autoGen-ipc")
	log.Noticef("Starting opera ipc %v", cfg.Db)

	resChan := make(chan string, 100)
	go func() {
		defer close(errChan)

		//cleanup opera.ipc when node is stopped
		defer func(name string) {
			err := os.Remove(name)
			if !os.IsNotExist(err) && err != nil {
				log.Errorf("failed to remove ipc file %s; %v", name, err)
			}
		}(cfg.Db + "/opera.ipc")

		cmd := exec.Command(getOperaBinary(cfg), "--datadir", cfg.Db, "--maxpeers=0")
		err := runCommand(cmd, resChan, stopChan, log)
		if err != nil {
			errChan <- fmt.Errorf("unable run ipc opera --datadir %v; binary %v; %v", cfg.Db, getOperaBinary(cfg), err)
		}
	}()

	log.Noticef("Waiting for ipc to start")
	errChanParser := make(chan error, 1)

	// wait for ipc to start
	waitDuration := 5 * time.Minute
	timer := time.NewTimer(waitDuration)

ipcLoadingProcessWait:
	for {
		select {
		// since resChan was used the output still needs to be read to prevent deadlock by chan being full
		case res, ok := <-resChan:
			if ok {
				// waiting for opera message in output which indicates that ipc is ready for usage
				if strings.Contains(res, "IPC endpoint opened") {
					log.Noticef(res)
					break ipcLoadingProcessWait
				}
			}
		case err, ok := <-errChan:
			if ok {
				// error happened, the opera ipc didn't start properly
				errChanParser <- fmt.Errorf("opera error during ipc initialization; %v", err)
			}
			// errChan closed, this means that stopChan signal was called to terminate opera ipc,
			// which otherwise without an error never stops on its own
			close(errChanParser)
			return errChanParser
		case <-timer.C:
			// if ipc didn't start in given time produce an error
			errChanParser <- fmt.Errorf("timeout waiting for opera ipc to start after %s", waitDuration.String())
			close(errChanParser)
			return errChanParser
		}
	}

	// non-blocking error relaying while reading from resChan to prevent deadlock
	go func() {
		defer close(errChanParser)
		for {
			select {
			// since resChan was used the output still needs to be read to prevent deadlock by chan being full
			case <-resChan:
			case err, ok := <-errChan:
				if ok {
					// error happened, the opera failed after ipc initialization
					errChanParser <- fmt.Errorf("opera error after ipc initialization; %v", err)
				}
				return
			}
		}
	}()

	return errChanParser
}

// startOperaRecording records substates
func startOperaRecording(cfg *utils.Config, syncUntilEpoch uint64) chan error {
	errChan := make(chan error, 1)
	// todo check if path to aidaDb exists otherwise create the dir

	log := logger.NewLogger(cfg.LogLevel, "autogen-recording")
	log.Noticef("Starting opera recording %v", cfg.Db)

	go func() {
		defer close(errChan)

		// syncUntilEpoch +1 because command is off by one
		cmd := exec.Command(getOperaBinary(cfg), "--datadir", cfg.Db, "--recording", "--substate-db", cfg.SubstateDb, "--exitwhensynced.epoch", strconv.FormatUint(syncUntilEpoch+1, 10))
		err := runCommand(cmd, nil, nil, log)
		if err != nil {
			errChan <- fmt.Errorf("unable to record opera substates %v; binary %v ; %v", cfg.Db, getOperaBinary(cfg), err)
		}
	}()
	return errChan
}

// getOperaBinary returns path to opera binary
func getOperaBinary(cfg *utils.Config) string {
	var operaBin = "opera"
	if cfg.OperaBinary != "" {
		operaBin = cfg.OperaBinary
	}
	return operaBin
}
