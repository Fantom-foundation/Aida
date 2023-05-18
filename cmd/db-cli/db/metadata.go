package db

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	MetadataPrefix   = "md"
	TimestampPrefix  = MetadataPrefix + "ti"
	FirstBlockPrefix = MetadataPrefix + "fi"
	LastBlockPrefix  = MetadataPrefix + "la"
)

// createMetadata and put it into db
func createMetadata(targetDb ethdb.Database, blockStart, blockEnd uint64) error {
	createTime := make([]byte, 8)
	binary.BigEndian.PutUint64(createTime, uint64(time.Now().UTC().Second()))
	if err := targetDb.Put([]byte(TimestampPrefix), createTime); err != nil {
		return fmt.Errorf("cannot put timestamp into db metadata; %v", err)
	}

	firstBlock := make([]byte, 8)
	binary.BigEndian.PutUint64(firstBlock, blockStart)
	if err := targetDb.Put([]byte(FirstBlockPrefix), firstBlock); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	lastBlock := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlock, blockEnd)
	if err := targetDb.Put([]byte(LastBlockPrefix), lastBlock); err != nil {
		return fmt.Errorf("cannot put last block number into db metadata; %v", err)
	}

	return nil
}
