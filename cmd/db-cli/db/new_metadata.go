package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
)

// Metadata holds any information about AidaDb needed for putting it into the Db
type Metadata struct {
	aidaDb                ethdb.Database
	dbType                aidaDbType
	log                   *logging.Logger
	chainId               int
	firstBlock, lastBlock uint64
	firstEpoch, lastEpoch uint64
}

func newProcessMetadata(aidaDb ethdb.Database, log *logging.Logger, chainID int, firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64) error {

	m := Metadata{
		aidaDb:     aidaDb,
		dbType:     genType,
		log:        log,
		chainId:    chainID,
		firstBlock: firstBlock,
		lastBlock:  lastBlock,
		firstEpoch: firstEpoch,
		lastEpoch:  lastEpoch,
	}

	m.log.Notice("Writing metadata...")
	defer m.log.Notice("Metadata written successfully")

	return m.process()
}

func (m *Metadata) process() error {
	switch m.dbType {
	case updateType:
		fallthrough
	case patchType:
		fallthrough
	case genType:
		return m.genMetadata()
	case mergeType:
		panic("not implemented yet")
	case cloneType:
		// clone type already has every metadata needed
		return nil
	default:

		return errors.New("unknown db type")

	}

}

func (m *Metadata) genMetadata() error {
	var err error

	if err = m.doBlocks(); err != nil {
		return err
	}

	if err = m.doEpochs(); err != nil {
		return err
	}

	byteChainID := bigendian.Uint16ToBytes(uint16(m.chainId))

	if err = m.aidaDb.Put([]byte(ChainIDPrefix), byteChainID); err != nil {
		return fmt.Errorf("cannot put chain-id into aida-db; %v", err)
	}

	// todo do we want to keep original timestamp or overwrite when merging, tapping etc...
	createTime := make([]byte, 8)

	binary.BigEndian.PutUint64(createTime, uint64(time.Now().Unix()))
	if err = m.aidaDb.Put([]byte(TimestampPrefix), createTime); err != nil {
		return fmt.Errorf("cannot put timestamp into aida-db; %v", err)
	}

	dbTypeBytes := make([]byte, 1)
	dbTypeBytes[0] = byte(m.dbType)

	if err = m.aidaDb.Put([]byte(TypePrefix), dbTypeBytes); err != nil {
		return fmt.Errorf("cannot put db-type into aida-db; %v", err)
	}

	return nil
}

func (m *Metadata) doBlocks() error {
	var (
		first, last                     uint64
		firstBlockBytes, lastBlockBytes []byte
		err                             error
	)

	firstBlockBytes, err = m.aidaDb.Get([]byte(FirstBlockPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get first block ; %v", err)
		}
		return nil
	}

	first = bigendian.BytesToUint64(firstBlockBytes)

	if first < m.firstBlock {
		m.firstBlock = first
	}

	lastBlockBytes, err = m.aidaDb.Get([]byte(LastBlockPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get last block; %v", err)
		}
		return nil
	}

	last = bigendian.BytesToUint64(lastBlockBytes)

	if last > m.lastBlock {
		m.lastBlock = last
	}

	return nil
}

func (m *Metadata) doEpochs() error {
	var (
		originalFirst, originalLast uint64
		writeFirst, writeLast       bool
	)

	// start with finding whether we are creating new AidaDb or no

	firstEpochBytes, err := m.aidaDb.Get([]byte(FirstEpochPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get originalFirst epoch ; %v", err)
		}
		return nil
	}

	originalFirst = bigendian.BytesToUint64(firstEpochBytes)

	// do we even need to write?
	if m.firstEpoch < originalFirst {
		writeFirst = true
	} else {
		m.firstEpoch = originalFirst
	}

	lastEpochBytes, err := m.aidaDb.Get([]byte(LastEpochPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get originalLast epoch; %v", err)
		}
		return nil
	}

	originalLast = bigendian.BytesToUint64(lastEpochBytes)

	// do we even need to write?
	if m.lastEpoch > originalLast {
		writeLast = true
	} else {
		m.lastEpoch = originalLast
	}

	if writeFirst {
		if err = m.writeFirstEpoch(); err != nil {
			return err
		}
	}

	if writeLast {
		if err = m.writeLastEpoch(); err != nil {
			return err
		}
	}

	return nil
}

func (m *Metadata) writeFirstEpoch() error {
	if m.firstEpoch == 0 {
		m.log.Warning("given first epoch is 0 - saving to metadata anyway")
	}

	firstEpochBytes := substate.BlockToBytes(m.firstEpoch)
	if err := m.aidaDb.Put([]byte(FirstEpochPrefix), firstEpochBytes); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	return nil
}

func (m *Metadata) writeLastEpoch() error {
	if m.lastEpoch == 0 {
		m.log.Warning("given last epoch is 0 - saving to metadata anyway")
	}

	// if db is type of clone, epochs are set to 0
	if m.dbType != cloneType {
		m.lastEpoch -= 1
	}

	lastEpochBytes := substate.BlockToBytes(m.lastEpoch)
	if err := m.aidaDb.Put([]byte(LastEpochPrefix), lastEpochBytes); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	return nil
}

// getLastBlock retrieve last block from aida-db Metadata
func getLastBlock(aidaDb ethdb.Database) (uint64, error) {
	lastBlockBytes, err := aidaDb.Get([]byte(LastBlockPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get last block from db; %v", err)
	}
	return bigendian.BytesToUint64(lastBlockBytes), nil
}
