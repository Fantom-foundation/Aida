package db

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
)

type Metadata interface {
	getFirstBlock() uint64
	getLastBlock() uint64
	getFirstEpoch() uint64
	getLastEpoch() uint64
	getChainID() int
	getTimestamp() uint64

	setFirstBlock(uint64)
	setLastBlock(uint64)
	setFirstEpoch(uint64)
	setLastEpoch(uint64)
	setChainID(int)
	setTimestamp()
}

// aidaMetadata holds any information about AidaDb needed for putting it into the Db
type aidaMetadata struct {
	aidaDb                                       ethdb.Database
	dbType                                       aidaDbType
	log                                          *logging.Logger
	firstBlock, lastBlock, firstEpoch, lastEpoch uint64
	chainId                                      int
}

func newProcessMetadata(aidaDb ethdb.Database, log *logging.Logger, chainID int, firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64) {
	m := aidaMetadata{
		aidaDb: aidaDb,
		dbType: genType,
		log:    log,
	}

	m.log.Notice("Writing metadata...")

	switch m.dbType {
	case updateType:
		fallthrough
	case patchType:
		fallthrough
	case genType:
		m.genMetadata(firstBlock, lastBlock, firstEpoch, lastEpoch, chainID)
		m.log.Notice("Metadata written successfully")
		return
	case mergeType:
		panic("not implemented yet")
	case cloneType:
		// clone type already has every metadata needed
		return
	default:
		log.Warningf("Unknown data type: %v", m.dbType)
	}

}

func (m *aidaMetadata) genMetadata(firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int) {
	m.doBlocks(firstBlock, lastBlock)

	m.doEpochs(firstEpoch, lastEpoch)

	m.setChainID(chainID)

	m.setDbType(m.dbType)

	m.setTimestamp()

}

func (m *aidaMetadata) doBlocks(firstBlock uint64, lastBlock uint64) {
	var (
		originalFirst, originalLast uint64
	)

	originalFirst = m.getFirstBlock()

	if originalFirst > firstBlock {
		m.setFirstBlock(firstBlock)
	}

	originalLast = m.getLastBlock()

	if originalLast != 0 && originalLast < lastBlock {
		m.setLastBlock(lastBlock)
	}
}

func (m *aidaMetadata) doEpochs(firstEpoch uint64, lastEpoch uint64) {
	var (
		originalFirst, originalLast uint64
	)

	originalFirst = m.getFirstEpoch()

	if originalFirst == 0 || originalFirst > firstEpoch {
		m.setFirstEpoch(firstEpoch)
	}

	originalLast = m.getLastEpoch()

	if originalLast == 0 || originalLast < lastEpoch {
		m.setLastEpoch(lastEpoch)
	}
}

func (m *aidaMetadata) getFirstBlock() uint64 {
	firstBlockBytes, err := m.aidaDb.Get([]byte(FirstBlockPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get first block; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(firstBlockBytes)
}

func (m *aidaMetadata) getLastBlock() uint64 {
	lastBlockBytes, err := m.aidaDb.Get([]byte(LastBlockPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get last block; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(lastBlockBytes)
}

func (m *aidaMetadata) getFirstEpoch() uint64 {
	firstEpochBytes, err := m.aidaDb.Get([]byte(FirstEpochPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get first epoch; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(firstEpochBytes)
}

func (m *aidaMetadata) getLastEpoch() uint64 {
	lastEpochBytes, err := m.aidaDb.Get([]byte(LastEpochPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get last epoch; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(lastEpochBytes)
}

func (m *aidaMetadata) getChainID() int {
	chainIDBytes, err := m.aidaDb.Get([]byte(ChainIDPrefix))
	if err != nil {
		m.log.Errorf("cannot get chain-id; %v", err)
		return 0
	}

	return int(bigendian.BytesToUint16(chainIDBytes))
}

func (m *aidaMetadata) getTimestamp() uint64 {
	byteChainID, err := m.aidaDb.Get([]byte(TimestampPrefix))
	if err != nil {
		m.log.Errorf("cannot get chain-id; %v", err)
		return 0
	}

	return bigendian.BytesToUint64(byteChainID)
}

func (m *aidaMetadata) setFirstBlock(firstBlock uint64) {
	firstBlockBytes := substate.BlockToBytes(firstBlock)

	if err := m.aidaDb.Put([]byte(FirstBlockPrefix), firstBlockBytes); err != nil {
		m.log.Errorf("cannot put first block; %v", err)
	}
}

func (m *aidaMetadata) setLastBlock(lastBlock uint64) {
	lastBlockBytes := substate.BlockToBytes(lastBlock)

	if err := m.aidaDb.Put([]byte(LastBlockPrefix), lastBlockBytes); err != nil {
		m.log.Errorf("cannot put last block; %v", err)
	}
}

func (m *aidaMetadata) setFirstEpoch(firstEpoch uint64) {
	firstEpochBytes := substate.BlockToBytes(firstEpoch)

	if err := m.aidaDb.Put([]byte(FirstEpochPrefix), firstEpochBytes); err != nil {
		m.log.Errorf("cannot put first epoch; %v", err)
	}
}

func (m *aidaMetadata) setLastEpoch(lastEpoch uint64) {
	lastEpochBytes := substate.BlockToBytes(lastEpoch)

	if err := m.aidaDb.Put([]byte(LastEpochPrefix), lastEpochBytes); err != nil {
		m.log.Errorf("cannot put last epoch; %v", err)
	}
}

func (m *aidaMetadata) setChainID(chainID int) {
	chainIDBytes := bigendian.Uint16ToBytes(uint16(chainID))

	if err := m.aidaDb.Put([]byte(ChainIDPrefix), chainIDBytes); err != nil {
		m.log.Errorf("cannot put chain-id; %v", err)
	}
}

func (m *aidaMetadata) setTimestamp() {
	createTime := make([]byte, 8)

	binary.BigEndian.PutUint64(createTime, uint64(time.Now().Unix()))
	if err := m.aidaDb.Put([]byte(TimestampPrefix), createTime); err != nil {
		m.log.Errorf("cannot put timestamp into db metadata; %v", err)
	}
}

func (m *aidaMetadata) setDbType(dbType aidaDbType) {
	dbTypeBytes := make([]byte, 1)
	dbTypeBytes[0] = byte(dbType)

	if err := m.aidaDb.Put([]byte(TypePrefix), dbTypeBytes); err != nil {
		m.log.Errorf("cannot put db-type into aida-db; %v", err)
	}
}

// getLastBlock retrieve last block from aida-db aidaMetadata
func getLastBlock(aidaDb ethdb.Database) (uint64, error) {
	lastBlockBytes, err := aidaDb.Get([]byte(LastBlockPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get last block from db; %v", err)
	}
	return bigendian.BytesToUint64(lastBlockBytes), nil
}
