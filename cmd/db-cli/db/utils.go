package db

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
)

// prepareDbDirs updates config for flags required in invoked generation commands
// these flags are not expected from user, so we need to specify them for the generation process
func prepareDbDirs(cfg *utils.Config) (string, error) {
	if cfg.DbTmp != "" {
		// create a parents of temporary directory
		err := os.MkdirAll(cfg.DbTmp, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
		}
	}

	//fName := fmt.Sprintf("%v/%v-%v", cfg.DbTmp, "tmp_aida_db_*", rand.Int())
	// create a temporary working directory
	fName, err := os.MkdirTemp(cfg.DbTmp, "aida_db_tmp_*")
	if err != nil {
		return "", fmt.Errorf("failed to create a temporary directory. %v", err)
	}

	loadSourceDBPaths(cfg, fName)

	return fName, nil
}

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

// loadSourceDBPaths initializes paths to source databases
func loadSourceDBPaths(cfg *utils.Config, aidaDbTmp string) {
	cfg.DeletionDb = filepath.Join(aidaDbTmp, "deletion")
	cfg.SubstateDb = filepath.Join(aidaDbTmp, "substate")
	cfg.UpdateDb = filepath.Join(aidaDbTmp, "update")
	cfg.WorldStateDb = filepath.Join(aidaDbTmp, "worldstate")
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
	defer close(lastOutputMessagesChan)
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

// startDaemonOpera start opera node
func startDaemonOpera(log *logging.Logger) error {
	cmd := exec.Command("systemctl", "--user", "start", "opera")
	err := runCommand(cmd, nil, log)
	if err != nil {
		return fmt.Errorf("unable start opera; %v", err.Error())
	}
	return nil
}

// stopDaemonOpera stop opera node
func stopDaemonOpera(log *logging.Logger) error {
	cmd := exec.Command("systemctl", "--user", "stop", "opera")
	err := runCommand(cmd, nil, log)
	if err != nil {
		return fmt.Errorf("unable stop opera; %v", err.Error())
	}
	return nil
}

// startOperaPruning prunes opera in parallel
func startOperaPruning(cfg *utils.Config) chan error {
	errChan := make(chan error, 1)

	log := logger.NewLogger(cfg.LogLevel, "autoGen-pruning")
	log.Noticef("Starting opera pruning %v", cfg.Db)

	go func() {
		defer close(errChan)
		cmd := exec.Command("opera", "--datadir", cfg.Db, "snapshot", "prune-state")
		err := runCommand(cmd, nil, log)
		if err != nil {
			errChan <- fmt.Errorf("unable prune opera %v; %v", cfg.Db, err)
		}
	}()
	return errChan
}
