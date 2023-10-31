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

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
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
func runCommand(cmd *exec.Cmd, resultChan chan string, log *logging.Logger) error {
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
	for scanner.Scan() {
		m := scanner.Text()
		if resultChan != nil {
			resultChan <- m
		}
		if log.IsEnabledFor(logging.DEBUG) {
			log.Debug(m)
		} else {
			// in case debugging is turned off and resultChan doesn't listen to output
			// we need to keep most recent output lines in case of error
			if resultChan == nil {
				// throw out the oldest line in case we are at limit
				if len(lastOutputMessagesChan) == commandOutputLimit {
					<-lastOutputMessagesChan
				}
				lastOutputMessagesChan <- m
			}
		}
	}
	close(lastOutputMessagesChan)
	err = cmd.Wait()

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
		err := runCommand(cmd, nil, log)
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
