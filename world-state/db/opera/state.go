// Package opera implements Opera specific database interfaces for the world state manager.
package opera

import (
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

// BlockEpochState provides joined block and epoch state stored in the provided Opera database.
func BlockEpochState(s kvdb.Store) (*types.BlockEpochState, error) {
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
func LatestStateRoot(s kvdb.Store) (common.Hash, error) {
	bes, err := BlockEpochState(s)
	if err != nil {
		return common.Hash{}, err
	}
	return bes.BlockState.FinalizedStateRoot, nil
}
