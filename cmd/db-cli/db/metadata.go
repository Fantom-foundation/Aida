package db

import (
	"encoding/binary"
	"fmt"
	"time"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	TimestampPrefix  = substate.MetadataPrefix + "ti"
	FirstBlockPrefix = substate.MetadataPrefix + "fi"
	LastBlockPrefix  = substate.MetadataPrefix + "la"
)

// createMetadata and put it into db
func createMetadata(targetDb ethdb.Database, blockStart, blockEnd uint64) error {
	createTime := make([]byte, 8)
	binary.BigEndian.PutUint64(createTime, uint64(time.Now().UTC().Second()))
	if err := targetDb.Put([]byte(TimestampPrefix), createTime); err != nil {
		return fmt.Errorf("cannot put timestamp into db metadata; %v", err)
	}

	firstBlock := substate.BlockToBytes(blockStart)
	binary.BigEndian.PutUint64(firstBlock, blockStart)
	if err := targetDb.Put([]byte(FirstBlockPrefix), firstBlock); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	lastBlock := substate.BlockToBytes(blockEnd)
	if err := targetDb.Put([]byte(LastBlockPrefix), lastBlock); err != nil {
		return fmt.Errorf("cannot put last block number into db metadata; %v", err)
	}

	return nil
}
