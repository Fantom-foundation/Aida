// Package opera implements Opera specific database interfaces for the world state manager.
package opera

import (
	"encoding/binary"
	"fmt"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// OpenBlockEpochState opens the Opera block/epoch state database.
func OpenBlockEpochState(store kvdb.Store) kvdb.Store {
	return table.New(store, []byte(("D")))
}

// blockEpochState provides joined block and epoch state stored in the provided Opera database.
func blockEpochState(s kvdb.Store) (*types.BlockEpochState, error) {
	ebs := OpenBlockEpochState(s)

	data, err := ebs.Get([]byte("s"))
	if err != nil {
		return nil, fmt.Errorf("block state not found; %s", err.Error())
	}

	bes := types.BlockEpochState{}
	err = rlp.DecodeBytes(data, &bes)
	if err != nil {
		return nil, fmt.Errorf("could not decode block/epoch state information; %s", err.Error())
	}

	return &bes, nil
}

// LatestStateRoot provides the latest block state root hash.
func LatestStateRoot(s kvdb.Store) (common.Hash, uint64, error) {
	bes, err := blockEpochState(s)
	if err != nil {
		return common.Hash{}, 0, err
	}
	return bes.BlockState.FinalizedStateRoot, bes.BlockState.LastBlock.Idx, nil
}

// OpenBlocks opens the Opera blocks database.
func OpenBlocks(store kvdb.Store) kvdb.Store {
	return table.New(store, []byte(("b")))
}

// RootBlock provides the block number for given root hash.
func RootBlock(s kvdb.Store, root common.Hash) (chan uint64, chan error) {
	blockNumberChan := make(chan uint64, 1)
	err := make(chan error, 1)

	go rootBLock(s, root, blockNumberChan, err)

	return blockNumberChan, err
}

// rootBLock iterate the blocks to find block with given root hash
func rootBLock(s kvdb.Store, root common.Hash, blockNumberChan chan uint64, fail chan error) {
	defer func() {
		close(blockNumberChan)
		close(fail)
	}()

	lastStateRoot, lastBlock, err := LatestStateRoot(s)
	if err != nil {
		fail <- fmt.Errorf("Last state root of database not found;  %s", root)
		return
	}

	// database doesn't have information recent information about blocks
	if lastBlock == 0 {
		fail <- fmt.Errorf("Last state root of database returned %d;  %s", lastBlock, root)
		return
	}

	// searched root is from last block
	if root == lastStateRoot {
		blockNumberChan <- lastBlock
		return
	}

	ebs := OpenBlocks(s)
	b := make([]byte, 8)

	for i := lastBlock; i >= 0; i-- {
		binary.BigEndian.PutUint64(b, i)
		data, err := ebs.Get(b)
		if err != nil {
			fail <- fmt.Errorf("block %d info not found; %s", i, err.Error())
			return
		}

		block := types.Block{}
		err = rlp.DecodeBytes(data, &block)
		if err != nil {
			fail <- fmt.Errorf("could not decode block %d information; %s", i, err.Error())
			return
		}

		// block with current index has matching root
		if common.Hash(block.Root) == root {
			// successfully found block number
			blockNumberChan <- i
			return
		}
	}

	fail <- fmt.Errorf("block for root %s was not found in database", root)
}
