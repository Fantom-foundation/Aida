package db

import (
	"encoding/binary"
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

type aidaDbType byte

const (
	noType aidaDbType = iota
	genType
	patchType
	cloneType
	updateType
)

const (
	FirstBlockPrefix = substate.MetadataPrefix + "fb"
	LastBlockPrefix  = substate.MetadataPrefix + "lb"
	FirstEpochPrefix = substate.MetadataPrefix + "fe"
	LastEpochPrefix  = substate.MetadataPrefix + "le"
	TypePrefix       = substate.MetadataPrefix + "ty"
	ChainIDPrefix    = substate.MetadataPrefix + "ci"
	TimestampPrefix  = substate.MetadataPrefix + "ti"
)

// merge is determined by what are we merging
// genType + cloneType / cloneType + cloneType / = NOT POSSIBLE
// genType + genType = genType
// genType + patchType = genType
// cloneType + patchType = cloneType
// patchType + patchType = patchType

// aidaMetadata holds any information about AidaDb needed for putting it into the Db
type aidaMetadata struct {
	db                    ethdb.Database
	log                   *logging.Logger
	firstBlock, lastBlock uint64
	firstEpoch, lastEpoch uint64
	chainId               int
	dbType                aidaDbType
	timestamp             uint64
}

// todo we need to check block alignment and chainID match before any merging

// newAidaMetadata creates new instance of aidaMetadata
func newAidaMetadata(db ethdb.Database, logLevel string) *aidaMetadata {
	return &aidaMetadata{
		db:  db,
		log: logger.NewLogger(logLevel, "aida-metadata"),
	}
}

// processPatchLikeMetadata decides whether patch is new or not. If so the dbType is set to genType, otherwise its patchType.
// Then it inserts all given metadata
func processPatchLikeMetadata(aidaDb ethdb.Database, logLevel string, firstBlock, lastBlock, firstEpoch, lastEpoch uint64, chainID int, isNew bool) {
	var dbType aidaDbType

	// if this is brand-new patch, it should be treated as a gen type db
	if isNew {
		dbType = genType
	} else {
		dbType = patchType
	}

	m := newAidaMetadata(aidaDb, logLevel)

	m.setFirstBlock(firstBlock)
	m.setLastBlock(lastBlock)

	m.setFirstEpoch(firstEpoch)
	m.setLastEpoch(lastEpoch)

	m.setChainID(chainID)

	m.setDbType(dbType)

	m.setTimestamp()

	m.log.Notice("Metadata added successfully")
}

// processCloneLikeMetadata inserts every metadata from sourceDb, only epochs are excluded.
// We can't be certain if given epoch is whole
func processCloneLikeMetadata(aidaDb ethdb.Database, logLevel string, firstBlock, lastBlock uint64, chainID int) {
	m := newAidaMetadata(aidaDb, logLevel)

	firstBlock, lastBlock = m.findBlocks(firstBlock, lastBlock)
	m.setFirstBlock(firstBlock)
	m.setLastBlock(lastBlock)

	m.setChainID(chainID)

	m.setDbType(cloneType)

	m.setTimestamp()

	m.log.Notice("Metadata added successfully")
}

// processGenLikeMetadata inserts metadata into newly generated AidaDb.
// If generate is used onto an existing AidaDb it updates last block, last epoch and timestamp.
func processGenLikeMetadata(aidaDb ethdb.Database, logLevel string, firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int) {
	m := newAidaMetadata(aidaDb, logLevel)

	firstBlock, lastBlock = m.findBlocks(firstBlock, lastBlock)
	m.setFirstBlock(firstBlock)
	m.setLastBlock(lastBlock)

	firstEpoch, lastEpoch = m.findEpochs(firstEpoch, lastEpoch)
	m.setFirstEpoch(firstEpoch)
	m.setLastEpoch(lastEpoch)

	m.setChainID(chainID)

	m.setDbType(genType)

	m.setTimestamp()

}

// processMergeMetadata decides the type according to the types of merged Dbs and inserts every metadata
func processMergeMetadata(aidaDb ethdb.Database, sourceDbs []ethdb.Database, logLevel string) {
	var (
		dbType                = patchType
		t                     aidaDbType
		firstBlock, lastBlock uint64
		firstEpoch, lastEpoch uint64
		chainID               int
	)

	for _, db := range sourceDbs {
		m := newAidaMetadata(db, logLevel)
		m.getMetadata()

		// todo do we need to check whether blocks align?

		// get chainID of first merged db
		if chainID == 0 {
			chainID = m.chainId
		}

		// if chain ids doesn't match, we should not be merging
		if m.chainId != chainID {
			m.log.Critical("ChainIDs in Dbs metadata does not match!")
		}

		if m.firstBlock < firstBlock || firstBlock == 0 {
			firstBlock = m.firstEpoch
		}

		if m.lastEpoch > lastBlock || lastBlock == 0 {
			lastBlock = m.lastBlock
		}

		t = m.dbType
		if t == cloneType {
			dbType = t
		} else if t == genType && dbType != cloneType {
			dbType = t
		}

	}

	aidaDbMetadata := newAidaMetadata(aidaDb, logLevel)

	aidaDbMetadata.setAllMetadata(firstBlock, lastBlock, firstEpoch, lastEpoch, chainID, dbType)

}

// getMetadata from given db and save it
func (m *aidaMetadata) getMetadata() {
	m.firstBlock = m.getFirstBlock()
	m.lastBlock = m.getLastBlock()
	m.firstEpoch = m.getFirstEpoch()
	m.lastEpoch = m.getLastEpoch()
	m.dbType = m.getDbType()
	m.timestamp = m.getTimestamp()
	m.chainId = m.getChainID()

	return
}

// findBlocks from given db and return them
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

// findEpochs from given db and return them
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

// getFirstBlock and return it
func (m *aidaMetadata) getFirstBlock() uint64 {
	firstBlockBytes, err := m.db.Get([]byte(FirstBlockPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get first block; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(firstBlockBytes)
}

// getLastBlock and return it
func (m *aidaMetadata) getLastBlock() uint64 {
	lastBlockBytes, err := m.db.Get([]byte(LastBlockPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get last block; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(lastBlockBytes)
}

// getFirstEpoch and return it
func (m *aidaMetadata) getFirstEpoch() uint64 {
	firstEpochBytes, err := m.db.Get([]byte(FirstEpochPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get first epoch; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(firstEpochBytes)
}

// getLastEpoch and return it
func (m *aidaMetadata) getLastEpoch() uint64 {
	lastEpochBytes, err := m.db.Get([]byte(LastEpochPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get last epoch; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(lastEpochBytes)
}

// getChainID and return it
func (m *aidaMetadata) getChainID() int {
	chainIDBytes, err := m.db.Get([]byte(ChainIDPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get chain-id; %v", err)
		}
		return 0
	}

	return int(bigendian.BytesToUint16(chainIDBytes))
}

// getTimestamp and return it
func (m *aidaMetadata) getTimestamp() uint64 {
	byteChainID, err := m.db.Get([]byte(TimestampPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get timestamp; %v", err)
		}
		return 0
	}

	return bigendian.BytesToUint64(byteChainID)
}

// getDbType and return it
func (m *aidaMetadata) getDbType() aidaDbType {
	byteDbType, err := m.db.Get([]byte(TypePrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			m.log.Errorf("cannot get db-type; %v", err)
		}
		return noType
	}

	return aidaDbType(byteDbType[0])
}

// setFirstBlock in given db
func (m *aidaMetadata) setFirstBlock(firstBlock uint64) {
	firstBlockBytes := substate.BlockToBytes(firstBlock)

	if err := m.db.Put([]byte(FirstBlockPrefix), firstBlockBytes); err != nil {
		m.log.Errorf("cannot put first block; %v", err)
	} else {
		m.log.Info("METADATA: First block saved successfully")
	}
}

// setLastBlock in given db
func (m *aidaMetadata) setLastBlock(lastBlock uint64) {
	lastBlockBytes := substate.BlockToBytes(lastBlock)

	if err := m.db.Put([]byte(LastBlockPrefix), lastBlockBytes); err != nil {
		m.log.Errorf("cannot put last block; %v", err)
	} else {
		m.log.Info("METADATA: Last block saved successfully")
	}
}

// setFirstEpoch in given db
func (m *aidaMetadata) setFirstEpoch(firstEpoch uint64) {
	firstEpochBytes := substate.BlockToBytes(firstEpoch)

	if err := m.db.Put([]byte(FirstEpochPrefix), firstEpochBytes); err != nil {
		m.log.Errorf("cannot put first epoch; %v", err)
	} else {
		m.log.Info("METADATA: First epoch saved successfully")
	}
}

// setLastEpoch in given db
func (m *aidaMetadata) setLastEpoch(lastEpoch uint64) {
	lastEpochBytes := substate.BlockToBytes(lastEpoch)

	if err := m.db.Put([]byte(LastEpochPrefix), lastEpochBytes); err != nil {
		m.log.Errorf("cannot put last epoch; %v", err)
	} else {
		m.log.Info("METADATA: Last epoch saved successfully")
	}
}

// setChainID in given db
func (m *aidaMetadata) setChainID(chainID int) {
	chainIDBytes := bigendian.Uint16ToBytes(uint16(chainID))

	if err := m.db.Put([]byte(ChainIDPrefix), chainIDBytes); err != nil {
		m.log.Errorf("cannot put chain-id; %v", err)
	} else {
		m.log.Info("METADATA: ChainID saved successfully")
	}
}

// setTimestamp in given db
func (m *aidaMetadata) setTimestamp() {
	createTime := make([]byte, 8)

	binary.BigEndian.PutUint64(createTime, uint64(time.Now().Unix()))
	if err := m.db.Put([]byte(TimestampPrefix), createTime); err != nil {
		m.log.Errorf("cannot put timestamp into db metadata; %v", err)
	} else {
		m.log.Info("METADATA: Creation timestamp saved successfully")
	}
}

// setDbType in given db
func (m *aidaMetadata) setDbType(dbType aidaDbType) {
	dbTypeBytes := make([]byte, 1)
	dbTypeBytes[0] = byte(dbType)

	if err := m.db.Put([]byte(TypePrefix), dbTypeBytes); err != nil {
		m.log.Errorf("cannot put db-type into aida-db; %v", err)
	} else {
		m.log.Info("METADATA: DB Type saved successfully")
	}
}

// setAllMetadata in given db
func (m *aidaMetadata) setAllMetadata(firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int, dbType aidaDbType) {
	m.setFirstBlock(firstBlock)

	m.setLastEpoch(lastBlock)

	m.setFirstEpoch(firstEpoch)

	m.setLastEpoch(lastEpoch)

	m.setChainID(chainID)

	m.setDbType(dbType)

	m.setTimestamp()
}
