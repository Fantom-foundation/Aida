package extension

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
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
	return &validateStateHashExtension{config: config, log: log}
}

type validateStateHashExtension struct {
	NilExtension
	config *utils.Config
	log    logger.Logger
	hashes []common.Hash
}

func (e *validateStateHashExtension) PreRun(executor.State) error {
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

func (e *validateStateHashExtension) PostBlock(state executor.State) error {
	if state.State == nil || state.Block >= len(e.hashes) {
		return nil
	}
	want := e.hashes[state.Block]
	got := state.State.GetHash()
	if want != got {
		return fmt.Errorf("unexpected hash for block %d\nwanted %v\n   got %v", state.Block, want, got)
	}
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
