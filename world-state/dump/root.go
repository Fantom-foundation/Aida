// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"encoding/binary"
	"github.com/Fantom-foundation/Aida-Testing/world-state/db"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// LatestStateRoot provides the latest block state root hash.
func LatestStateRoot(s kvdb.Store, log Logger) (common.Hash, uint64) {
	ebs := db.OpenBlockEpochState(s)

	data, err := ebs.Get([]byte("s"))
	if err != nil {
		log.Errorf("block state not found; %s", err.Error())
		return [32]byte{}, 0
	}

	state := types.BlockEpochState{}
	err = rlp.DecodeBytes(data, &state)
	if err != nil {
		log.Errorf("could not decode block/epoch state information; %s", err.Error())
	}

	log.Infof("Found Block: #%d", state.BlockState.LastBlock.Idx)
	log.Infof("Using State Root: %s", state.BlockState.FinalizedStateRoot.String())
	return state.BlockState.FinalizedStateRoot, state.BlockState.LastBlock.Idx
}

// RootBlock provides the block number for given root hash.
func RootBlock(s kvdb.Store, root common.Hash, log Logger) chan uint64 {
	blockNumberChan := make(chan uint64, 1)

	go RootBLock(s, root, blockNumberChan, log)

	return blockNumberChan
}

// RootBLock iterate the blocks to find block with given root hash
func RootBLock(s kvdb.Store, root common.Hash, blockNumberChan chan uint64, log Logger) {
	defer close(blockNumberChan)

	lastStateRoot, lastBlock := LatestStateRoot(s, log)

	// database doesn't have information recent information about blocks
	if lastBlock == 0 {
		blockNumberChan <- 0
		return
	}

	// searched root is from last block
	if root == lastStateRoot {
		blockNumberChan <- lastBlock
		return
	}

	ebs := db.OpenBlocks(s)
	b := make([]byte, 8)

	for i := lastBlock; i >= 0; i-- {
		binary.BigEndian.PutUint64(b, i)
		data, err := ebs.Get(b)
		if err != nil {
			log.Errorf("block %d info not found; %s", i, err.Error())
			break
		}

		block := types.Block{}
		err = rlp.DecodeBytes(data, &block)
		if err != nil {
			log.Errorf("could not decode block %d information; %s", i, err.Error())
			break
		}

		// block with current index has matching root
		if common.Hash(block.Root) == root {
			blockNumberChan <- i
			return
		}
	}

	log.Errorf("block for root %s was not found in database", root)
}
