// Package dump implements world state trie dump into a snapshot database.
package dump

import (
	"github.com/Fantom-foundation/Aida-Testing/world-state/db"
	"github.com/Fantom-foundation/Aida-Testing/world-state/types"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// LatestStateRoot provides the latest block state root hash.
func LatestStateRoot(s kvdb.Store, log Logger) common.Hash {
	ebs := db.OpenBlockEpochState(s)

	data, err := ebs.Get([]byte("s"))
	if err != nil {
		log.Errorf("block state not found; %s", err.Error())
		return [32]byte{}
	}

	state := types.BlockEpochState{}
	err = rlp.DecodeBytes(data, &state)
	if err != nil {
		log.Errorf("could not decode block/epoch state information; %s", err.Error())
	}

	log.Infof("Found Block: #%d", state.BlockState.LastBlock.Idx)
	log.Infof("Using State Root: %s", state.BlockState.FinalizedStateRoot.String())
	return state.BlockState.FinalizedStateRoot
}
