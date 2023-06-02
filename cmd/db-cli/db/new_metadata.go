package db

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
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
	getDbType() aidaDbType

	setFirstBlock(uint64)
	setLastBlock(uint64)
	setFirstEpoch(uint64)
	setLastEpoch(uint64)
	setChainID(int)
	setTimestamp()
	setDbType(aidaDbType)
}

// aidaMetadata holds any information about AidaDb needed for putting it into the Db
type aidaMetadata struct {
	aidaDb                                       ethdb.Database
	dbType                                       aidaDbType
	log                                          *logging.Logger
	firstBlock, lastBlock, firstEpoch, lastEpoch uint64
	chainId                                      int
}

func newAidaMetadata(db ethdb.Database, dbType aidaDbType, logLevel string) *aidaMetadata {
	return &aidaMetadata{
		aidaDb: db,
		dbType: dbType,
		log:    logger.NewLogger(logLevel, "aida-metadata"),
	}
}

func processGenLikeMetadata(aidaDb ethdb.Database, logLevel string, firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int) {
	m := newAidaMetadata(aidaDb, genType, logLevel)

	firstBlock, lastBlock = m.findBlocks(firstBlock, lastBlock)
	m.setFirstBlock(firstBlock)
	m.setLastBlock(lastBlock)

	firstEpoch, lastEpoch = m.findEpochs(firstEpoch, lastEpoch)
	m.setFirstEpoch(firstEpoch)
	m.setLastEpoch(lastEpoch)

	m.setChainID(chainID)

	m.setDbType(m.dbType)

	m.setTimestamp()

}

func processMergeMetadata(aidaDb ethdb.Database, sourceDbs []ethdb.Database, logLevel string) {
	var (
		dbType                = patchType
		t                     aidaDbType
		firstBlock, lastBlock uint64
		firstEpoch, lastEpoch uint64
		chainID               int
	)

	for _, db := range sourceDbs {
		m := newAidaMetadata(db, noType, logLevel)
		firstBlock, lastBlock = m.findBlocks(firstBlock, lastBlock)
		firstEpoch, lastEpoch = m.findEpochs(firstEpoch, lastEpoch)
		t = m.getDbType()
		if t == cloneType {
			dbType = t
		} else if t == genType && dbType != cloneType {
			dbType = t
		}

	}

	aidaDbMetadata := newAidaMetadata(aidaDb, dbType, logLevel)

	aidaDbMetadata.setFirstBlock(firstBlock)

	aidaDbMetadata.setLastEpoch(lastBlock)

	aidaDbMetadata.setFirstEpoch(firstEpoch)

	aidaDbMetadata.setLastEpoch(lastEpoch)

	aidaDbMetadata.setChainID(chainID)

	aidaDbMetadata.setDbType(aidaDbMetadata.dbType)

	aidaDbMetadata.setTimestamp()

}

func (m *aidaMetadata) getMetadata() (firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, dbType aidaDbType) {
	firstBlock = m.getFirstBlock()
	lastBlock = m.getLastBlock()
	firstEpoch = m.getFirstEpoch()
	lastEpoch = m.getLastEpoch()
	dbType = m.getDbType()

	return
}

func (m *aidaMetadata) findBlocks(firstBlock uint64, lastBlock uint64) (uint64, uint64) {
	var (
		dbFirst, dbLast uint64
	)

	dbFirst = m.getFirstBlock()

	if (dbFirst != 0 && dbFirst < firstBlock) || firstBlock == 0 {
		firstBlock = dbFirst
	}

	dbLast = m.getLastBlock()

	if dbLast > lastBlock || lastBlock == 0 {
		lastBlock = dbLast
	}

	return firstBlock, lastBlock
}

func (m *aidaMetadata) findEpochs(firstEpoch uint64, lastEpoch uint64) (uint64, uint64) {
	var (
		dbFirst, dbLast uint64
	)

	dbFirst = m.getFirstEpoch()

	if (dbFirst != 0 && dbFirst < firstEpoch) || firstEpoch == 0 {
		firstEpoch = dbFirst
	}

	dbLast = m.getLastEpoch()

	if dbLast > lastEpoch || lastEpoch == 0 {
		lastEpoch = dbLast
	}

	return firstEpoch, lastEpoch
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

func (m *aidaMetadata) getDbType() aidaDbType {
	byteDbType, err := m.aidaDb.Get([]byte(TypePrefix))
	if err != nil {
		m.log.Errorf("cannot get db-type; %v", err)
		return noType
	}

	return aidaDbType(byteDbType[0])
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

func (m *aidaMetadata) mergeMetadata(firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int, sourceDbs []ethdb.Database) {

}

// getLastBlock retrieve last block from aida-db aidaMetadata
func getLastBlock(aidaDb ethdb.Database) (uint64, error) {
	lastBlockBytes, err := aidaDb.Get([]byte(LastBlockPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get last block from db; %v", err)
	}
	return bigendian.BytesToUint64(lastBlockBytes), nil
}
