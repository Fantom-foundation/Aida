package extension

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
)

func MakeStateHashValidator(config *utils.Config) executor.Extension {
	log := logger.NewLogger("INFO", "state-hash-validator")
	return makeStateHashValidator(config, log)
}

func makeStateHashValidator(config *utils.Config, log logger.Logger) executor.Extension {
	if config.StateRootFile == "" {
		return NilExtension{}
	}
	return &stateHashValidator{config: config, log: log}
}

type stateHashValidator struct {
	NilExtension
	config                  *utils.Config
	log                     logger.Logger
	hashes                  []common.Hash
	nextArchiveBlockToCheck int
	lastProcessedBlock      int
}

func (e *stateHashValidator) PreRun(executor.State, *executor.Context) error {
	path := e.config.StateRootFile
	e.log.Infof("Loading state root hashes from %v ...", path)
	hashes, err := loadStateHashes(path, int(e.config.Last)+1)
	if err != nil {
		return err
	}
	e.hashes = hashes
	e.log.Infof("Loaded %d state root hashes from %v", len(e.hashes), path)
	return nil
}

func (e *stateHashValidator) PostBlock(state executor.State, context *executor.Context) error {
	if context.State == nil || state.Block >= len(e.hashes) {
		return nil
	}

	// Check the LiveDB
	want := e.hashes[state.Block]
	got := context.State.GetHash()
	if want != got {
		return fmt.Errorf("unexpected hash for Live block %d\nwanted %v\n   got %v", state.Block, want, got)
	}

	// Check the ArchiveDB
	if e.config.ArchiveMode {
		e.lastProcessedBlock = state.Block
		if err := e.checkArchiveHashes(context.State); err != nil {
			return err
		}
	}

	return nil
}

func (e *stateHashValidator) PostRun(state executor.State, context *executor.Context, err error) error {
	// Skip processing if run is aborted due to an error.
	if err != nil {
		return nil
	}
	// Complete processing remaining archive blocks.
	if e.config.ArchiveMode {
		for e.nextArchiveBlockToCheck < e.lastProcessedBlock {
			if err := e.checkArchiveHashes(context.State); err != nil {
				return err
			}
			if int(e.nextArchiveBlockToCheck) < e.lastProcessedBlock {
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
	return nil
}

func (e *stateHashValidator) checkArchiveHashes(state state.StateDB) error {
	// Note: the archive may be lagging behind the life DB, so block hashes need
	// to be checked as they become available.
	height, empty, err := state.GetArchiveBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get archive block height: %v", err)
	}

	cur := uint64(e.nextArchiveBlockToCheck)
	for !empty && cur <= height {

		archive, err := state.GetArchiveState(cur)
		if err != nil {
			return err
		}

		want := e.hashes[cur]
		got := archive.GetHash()
		if want != got {
			return fmt.Errorf("unexpected hash for Archive block %d\nwanted %v\n   got %v", cur, want, got)
		}

		cur++
	}
	e.nextArchiveBlockToCheck = int(cur)
	return nil
}

// loadStateHashes attempts to parse a file listing state roots in the format
//
//	(<block> - <hash>\n)*
//
// where <block> is a decimal block number and <hash> is a 64-character long,
// hexadecimal hash. Blocks are required to be listed in order, however, gaps
// may exist for blocks exhibiting the same hash as their predecessor.
// The limit parameter is the first block that is no longer loaded.
func loadStateHashes(path string, limit int) ([]common.Hash, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hashes := make([]common.Hash, 0, limit)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, " - ")
		if len(parts) != 2 || len(parts[1]) < 3 {
			return nil, fmt.Errorf("invalid line in hash list detected: `%s`", line)
		}

		block, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}

		if block < len(hashes) {
			return nil, fmt.Errorf("lines in state hash file are not sorted, encountered block %d after block %d", block, len(hashes)-1)
		}

		limitReached := false
		if block >= limit {
			block = limit
			limitReached = true
		}

		for len(hashes) < block {
			if len(hashes) == 0 {
				hashes = append(hashes, common.Hash{})
			} else {
				hashes = append(hashes, hashes[len(hashes)-1])
			}
		}

		if limitReached {
			break
		}

		bytes, err := hex.DecodeString(parts[1][2:])
		if err != nil {
			return nil, fmt.Errorf("unable to decode %s as hash value", parts[1][2:])
		}
		var hash common.Hash
		copy(hash[:], bytes)

		hashes = append(hashes, hash)
	}

	return hashes, nil
}
