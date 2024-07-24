// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utildb

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
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
	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/op/go-logging"
)

// OpenSourceDatabases opens all databases required for merge
func OpenSourceDatabases(sourceDbPaths []string) ([]db.BaseDB, error) {
	if len(sourceDbPaths) < 1 {
		return nil, fmt.Errorf("no source database were specified\n")
	}

	var sourceDbs []db.BaseDB
	for i := 0; i < len(sourceDbPaths); i++ {
		path := sourceDbPaths[i]
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source database %s; doesn't exist\n", path)
		}
		db, err := db.NewReadOnlyBaseDB(path)
		if err != nil {
			return nil, fmt.Errorf("source database %s; error: %v", path, err)
		}
		sourceDbs = append(sourceDbs, db)
	}

	return sourceDbs, nil
}

// MustCloseDB close database safely
func MustCloseDB(db db.BaseDB) {
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
func runCommand(cmd *exec.Cmd, resultChan chan string, stopChan chan struct{}, log logger.Logger) error {
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
			if ok {
				processScannedCommandOutput(m, resultChan, log, lastOutputMessagesChan)
				break
			}

			close(lastOutputMessagesChan)

			// wait until command finishes or stopSignal is received
			select {
			case <-stopChan:
				return killCommand(cmd, log, done)
			case res, ok := <-done:
				return processCommandResult(res, ok, scanner, lastOutputMessagesChan, resultChan, cmd, log)
			}
		}
	}
}

// killCommand terminates command gracefully first and then forcefully
func killCommand(cmd *exec.Cmd, log logger.Logger, done chan error) error {
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
func processScannedCommandOutput(message string, resultChan chan string, log logger.Logger, lastOutputMessagesChan chan string) {
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
func processCommandResult(err error, ok bool, scanner *bufio.Scanner, lastOutputMessagesChan chan string, resultChan chan string, cmd *exec.Cmd, log logger.Logger) error {
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

	log := logger.NewLogger(cfg.LogLevel, "Autogen-ipc")
	log.Noticef("Starting opera ipc %v", cfg.OperaDb)

	resChan := make(chan string, 100)
	go func() {
		defer close(errChan)

		//cleanup opera.ipc when node is stopped
		defer func(name string) {
			err := os.Remove(name)
			if !os.IsNotExist(err) && err != nil {
				log.Errorf("failed to remove ipc file %s; %v", name, err)
			}
		}(cfg.OperaDb + "/opera.ipc")

		cmd := exec.Command(getOperaBinary(cfg), "--datadir", cfg.OperaDb, "--maxpeers=0")
		err := runCommand(cmd, resChan, stopChan, log)
		if err != nil {
			errChan <- fmt.Errorf("unable run ipc opera --datadir %v; binary %v; %v", cfg.OperaDb, getOperaBinary(cfg), err)
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
	log.Noticef("Starting opera recording %v", cfg.OperaDb)

	go func() {
		defer close(errChan)

		// syncUntilEpoch +1 because command is off by one
		cmd := exec.Command(getOperaBinary(cfg), "--datadir", cfg.OperaDb, "--recording", "--substate-db", cfg.SubstateDb, "--exitwhensynced.epoch", strconv.FormatUint(syncUntilEpoch+1, 10))
		err := runCommand(cmd, nil, nil, log)
		if err != nil {
			errChan <- fmt.Errorf("unable to record opera substates %v; binary %v ; %v", cfg.OperaDb, getOperaBinary(cfg), err)
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

// GetDbSize retrieves database size
func GetDbSize(db db.BaseDB) uint64 {
	var count uint64
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		count++
	}
	return count
}

// PrintMetadata from given AidaDb
func PrintMetadata(pathToDb string) error {
	log := logger.NewLogger("INFO", "Print-Metadata")
	base, err := db.NewReadOnlyBaseDB(pathToDb)
	if err != nil {
		return err
	}

	md := utils.NewAidaDbMetadata(base, "INFO")

	log.Notice("AIDA-DB INFO:")

	if err = printDbType(md); err != nil {
		return err
	}

	lastBlock := md.GetLastBlock()

	firstBlock := md.GetFirstBlock()

	// CHAIN-ID
	chainID := md.GetChainID()

	if firstBlock == 0 && lastBlock == 0 && chainID == 0 {
		log.Error("your db does not contain metadata; please use metadata generate command")
	} else {
		log.Infof("Chain-ID: %v", chainID)

		// BLOCKS
		log.Infof("First Block: %v", firstBlock)

		log.Infof("Last Block: %v", lastBlock)

		// EPOCHS
		firstEpoch := md.GetFirstEpoch()

		log.Infof("First Epoch: %v", firstEpoch)

		lastEpoch := md.GetLastEpoch()

		log.Infof("Last Epoch: %v", lastEpoch)

		dbHash := md.GetDbHash()

		log.Infof("Db Hash: %v", hex.EncodeToString(dbHash))

		// TIMESTAMP
		timestamp := md.GetTimestamp()

		log.Infof("Created: %v", time.Unix(int64(timestamp), 0))
	}

	// UPDATE-SET
	printUpdateSetInfo(md)

	return nil
}

// printUpdateSetInfo from given AidaDb
func printUpdateSetInfo(m *utils.AidaDbMetadata) {
	log := logger.NewLogger("INFO", "Print-Metadata")

	log.Notice("UPDATE-SET INFO:")

	intervalBytes, err := m.Db.Get([]byte(db.UpdatesetIntervalKey))
	if err != nil {
		log.Warning("Value for update-set interval does not exist in given Dbs metadata")
	} else {
		log.Infof("Interval: %v blocks", bigendian.BytesToUint64(intervalBytes))
	}

	sizeBytes, err := m.Db.Get([]byte(db.UpdatesetSizeKey))
	if err != nil {
		log.Warning("Value for update-set size does not exist in given Dbs metadata")
	} else {
		u := bigendian.BytesToUint64(sizeBytes)

		log.Infof("Size: %.1f MB", float64(u)/float64(1_000_000))
	}
}

// printDbType from given AidaDb
func printDbType(m *utils.AidaDbMetadata) error {
	t := m.GetDbType()

	var typePrint string
	switch t {
	case utils.GenType:
		typePrint = "Generate"
	case utils.CloneType:
		typePrint = "Clone"
	case utils.PatchType:
		typePrint = "Patch"
	case utils.NoType:
		typePrint = "NoType"

	default:
		return errors.New("unknown db type")
	}

	logger.NewLogger("INFO", "Print-Metadata").Noticef("DB-Type: %v", typePrint)

	return nil
}

// LogDetailedSize counts and prints all prefix occurrence
func LogDetailedSize(db db.BaseDB, log logger.Logger) {
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	countMap := make(map[string]uint64)

	for iter.Next() {
		countMap[string(iter.Key()[:2])]++
	}

	for key, count := range countMap {
		log.Noticef("Prefix :%v; Count: %v", key, count)
	}
}
