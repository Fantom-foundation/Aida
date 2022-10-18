// Package opera implements Opera specific database interfaces for the world state manager.
package opera

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/Fantom-foundation/Aida/world-state/types"
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

// RootBLock iterate the blocks to find block with given root hash
func RootBLock(ctx context.Context, s kvdb.Store, root common.Hash) (uint64, error) {
	lastStateRoot, lastBlock, err := LatestStateRoot(s)
	if err != nil {
		return 0, fmt.Errorf("last state root of database not found;  %s", root)
	}

	// database doesn't have information recent information about blocks
	if lastBlock == 0 {
		return 0, fmt.Errorf("last state root of database returned %d;  %s", lastBlock, root)
	}

	// searched root is from last block
	if root == lastStateRoot {
		return lastBlock, nil
	}

	ebs := table.New(s, []byte(("b")))
	b := make([]byte, 8)

	ctxDone := ctx.Done()
	for i := lastBlock; i >= 0; i-- {
		select {
		case <-ctxDone:
			return 0, ctx.Err()
		default:
		}

		binary.BigEndian.PutUint64(b, i)
		data, err := ebs.Get(b)
		if err != nil {
			return 0, fmt.Errorf("block %d info not found; %s", i, err.Error())
		}

		block := types.Block{}
		err = rlp.DecodeBytes(data, &block)
		if err != nil {
			return 0, fmt.Errorf("could not decode block %d information; %s", i, err.Error())
		}

		// block with current index has matching root
		if common.Hash(block.Root) == root {
			// successfully found block number
			return i, nil
		}
	}

	return 0, fmt.Errorf("block for root %s was not found in database", root)
}

// RootOfBLock retrieves root hash from given block number
func RootOfBLock(s kvdb.Store, bn uint64) (common.Hash, error) {
	ebs := table.New(s, []byte(("b")))

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, bn)
	data, err := ebs.Get(key)
	if err != nil {
		return common.Hash{}, fmt.Errorf("block %d info not found in database; %s", bn, err.Error())
	}

	block := types.Block{}
	err = rlp.DecodeBytes(data, &block)
	if err != nil {
		return common.Hash{}, fmt.Errorf("could not decode block %d information; %s", bn, err.Error())
	}

	return common.Hash(block.Root), nil
}
