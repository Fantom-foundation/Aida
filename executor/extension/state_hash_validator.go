package extension

import (
	"bufio"
	"encoding/hex"
	"errors"
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
	"github.com/syndtr/goleveldb/leveldb"
)

func MakeStateHashValidator(config *utils.Config) executor.Extension {
	if !config.ValidateStateHashes {
		return NilExtension{}
	}

	log := logger.NewLogger("INFO", "state-hash-validator")
	return makeStateHashValidator(config, log)
}

func makeStateHashValidator(config *utils.Config, log logger.Logger) *stateHashValidator {
	return &stateHashValidator{config: config, log: log}
}

type stateHashValidator struct {
	NilExtension
	config                  *utils.Config
	log                     logger.Logger
	nextArchiveBlockToCheck int
	lastProcessedBlock      int
	hashProvider            utils.StateHashProvider
}

func (e *stateHashValidator) PreRun(_ executor.State, ctx *executor.Context) error {
	e.hashProvider = utils.MakeStateRootProvider(ctx.AidaDb)
	return nil
}

func (e *stateHashValidator) PostBlock(state executor.State, context *executor.Context) error {
	if context.State == nil {
		return nil
	}

	want, err := e.hashProvider.GetStateHash(state.Block)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			e.log.Warningf("State hash for block %v is not present in the db", state.Block)
			return nil
		}
		return fmt.Errorf("cannot get state hash for block %v; %v", state.Block, err)
	}

	got := context.State.GetHash()
	if want != got {
		return fmt.Errorf("unexpected hash for Live block %d\nwanted %v\n   got %v", state.Block, want, got)
	}

	// Check the ArchiveDB
	if e.config.ArchiveMode {
		e.lastProcessedBlock = state.Block
		if err = e.checkArchiveHashes(context.State); err != nil {
			return err
		}
	}

	return nil
}

func (e *stateHashValidator) PostRun(_ executor.State, context *executor.Context, err error) error {
	// Skip processing if run is aborted due to an error.
	if err != nil {
		return nil
	}
	// Complete processing remaining archive blocks.
	if e.config.ArchiveMode {
		for e.nextArchiveBlockToCheck < e.lastProcessedBlock {
			if err = e.checkArchiveHashes(context.State); err != nil {
				return err
			}
			if e.nextArchiveBlockToCheck < e.lastProcessedBlock {
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

		want, err := e.hashProvider.GetStateHash(int(cur))
		if err != nil {
			if errors.Is(err, leveldb.ErrNotFound) {
				e.log.Warningf("State hash for block %v is not present in the db", cur)
				return nil
			}
			return fmt.Errorf("cannot get state hash for block %v; %v", cur, err)
		}

		got := archive.GetHash()
		archive.Release()
		if want != got {
			return fmt.Errorf("unexpected hash for archive block %d\nwanted %v\n   got %v", cur, want, got)
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
