package db

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
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

const (
	RPCMainnet = "https://rpcapi.fantom.network"
	RPCTestnet = "https://rpc.testnet.fantom.network/"
)

type jsonRPCRequest struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      uint64        `json:"id"`
	JSONRPC string        `json:"jsonrpc"`
}

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
func processPatchLikeMetadata(aidaDb ethdb.Database, logLevel string, firstBlock, lastBlock, firstEpoch, lastEpoch uint64, chainID int, isNew bool) error {
	var (
		dbType aidaDbType
		err    error
	)

	// if this is brand-new patch, it should be treated as a gen type db
	if isNew {
		dbType = genType
	} else {
		dbType = patchType
	}

	md := newAidaMetadata(aidaDb, logLevel)

	if err = md.setFirstBlock(firstBlock); err != nil {
		return err
	}
	if err = md.setLastBlock(lastBlock); err != nil {
		return err
	}

	if err = md.setFirstEpoch(firstEpoch); err != nil {
		return err
	}
	if err = md.setLastEpoch(lastEpoch); err != nil {
		return err
	}

	if err = md.setChainID(chainID); err != nil {
		return err
	}

	if err = md.setDbType(dbType); err != nil {
		return err
	}

	if err = md.setTimestamp(); err != nil {
		return err
	}

	md.log.Notice("Metadata added successfully")

	return nil
}

// processCloneLikeMetadata inserts every metadata from sourceDb, only epochs are excluded.
// We can't be certain if given epoch is whole
func processCloneLikeMetadata(aidaDb ethdb.Database, logLevel string, firstBlock, lastBlock uint64, chainID int) error {
	var err error

	md := newAidaMetadata(aidaDb, logLevel)

	firstBlock, lastBlock, err = md.compareBlocks(firstBlock, lastBlock)
	if err != nil {
		return err
	}

	if err = md.setFirstBlock(firstBlock); err != nil {
		return err
	}
	if err = md.setLastBlock(lastBlock); err != nil {
		return err
	}

	if err = md.setChainID(chainID); err != nil {
		return err
	}

	if err = md.setDbType(cloneType); err != nil {
		return err
	}

	if err = md.setTimestamp(); err != nil {
		return err
	}

	md.log.Notice("Metadata added successfully")
	return nil
}

func processGenLikeMetadata(pathToAidaDb string, firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int, logLevel string) error {
	aidaDb, err := rawdb.NewLevelDBDatabase(pathToAidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("metadata cannot open AidaDb; %v", err)
	}

	defer MustCloseDB(aidaDb)

	md := newAidaMetadata(aidaDb, logLevel)
	return md.genMetadata(firstBlock, lastBlock, firstEpoch, lastEpoch, chainID)
}

// genMetadata inserts metadata into newly generated AidaDb.
// If generate is used onto an existing AidaDb it updates last block, last epoch and timestamp.
func (m *aidaMetadata) genMetadata(firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int) error {
	var err error

	firstBlock, lastBlock, err = m.compareBlocks(firstBlock, lastBlock)
	if err != nil {
		return err
	}

	if err = m.setFirstBlock(firstBlock); err != nil {
		return err
	}
	if err = m.setLastBlock(lastBlock); err != nil {
		return err
	}

	firstEpoch, lastEpoch, err = m.compareEpochs(firstEpoch, lastEpoch)
	if err != nil {
		return err
	}

	if err = m.setFirstEpoch(firstEpoch); err != nil {
		return err
	}
	if err = m.setLastEpoch(lastEpoch); err != nil {
		return err
	}

	if err = m.setChainID(chainID); err != nil {
		return err
	}

	if err = m.setDbType(genType); err != nil {
		return err
	}

	if err = m.setTimestamp(); err != nil {
		return err
	}

	return nil
}

// processMergeMetadata decides the type according to the types of merged Dbs and inserts every metadata
func processMergeMetadata(aidaDb ethdb.Database, sourceDbs []ethdb.Database, logLevel string) error {
	var (
		dbType                = patchType
		t                     aidaDbType
		firstBlock, lastBlock uint64
		firstEpoch, lastEpoch uint64
		chainID               int
		err                   error
	)

	for _, db := range sourceDbs {
		md := newAidaMetadata(db, logLevel)
		if err = md.getMetadata(); err != nil {
			return err
		}

		// todo do we need to check whether blocks align?

		// get chainID of first merged db
		if chainID == 0 {
			chainID = md.chainId
		}

		// if chain ids doesn't match, we should not be merging
		if md.chainId != chainID {
			md.log.Critical("ChainIDs in Dbs metadata does not match!")
		}

		if md.firstBlock < firstBlock || firstBlock == 0 {
			firstBlock = md.firstEpoch
		}

		if md.lastEpoch > lastBlock || lastBlock == 0 {
			lastBlock = md.lastBlock
		}

		t = md.dbType
		if t == cloneType {
			dbType = t
		} else if t == genType && dbType != cloneType {
			dbType = t
		}

	}

	md := newAidaMetadata(aidaDb, logLevel)

	return md.setAllMetadata(firstBlock, lastBlock, firstEpoch, lastEpoch, chainID, dbType)
}

// getMetadata from given db and save it
func (m *aidaMetadata) getMetadata() error {
	var err error

	m.firstBlock, err = m.getFirstBlock()
	if err != nil {
		return err
	}
	m.lastBlock, err = m.getLastBlock()
	if err != nil {
		return err
	}
	m.firstEpoch, err = m.getFirstEpoch()
	if err != nil {
		return err
	}
	m.lastEpoch, err = m.getLastEpoch()
	if err != nil {
		return err
	}
	m.dbType, err = m.getDbType()
	if err != nil {
		return err
	}
	m.timestamp, err = m.getTimestamp()
	if err != nil {
		return err
	}
	m.chainId, err = m.getChainID()
	if err != nil {
		return err
	}

	return nil
}

// compareBlocks from given db and return them
func (m *aidaMetadata) compareBlocks(firstBlock uint64, lastBlock uint64) (uint64, uint64, error) {
	var (
		dbFirst, dbLast uint64
		err             error
	)

	dbFirst, err = m.getFirstBlock()
	if err != nil {
		if strings.Contains(err.Error(), "leveldb: not found") {
			// block was not found, set to 0
			dbFirst = 0
		} else {
			return 0, 0, err
		}
	}

	if (dbFirst != 0 && dbFirst < firstBlock) || firstBlock == 0 {
		firstBlock = dbFirst
	}

	dbLast, err = m.getLastBlock()
	if err != nil {
		if strings.Contains(err.Error(), "leveldb: not found") {
			// block was not found, set to 0
			dbLast = 0
		} else {
			return 0, 0, err
		}
	}

	if dbLast > lastBlock || lastBlock == 0 {
		lastBlock = dbLast
	}

	return firstBlock, lastBlock, nil
}

// compareEpochs from given db and return them
func (m *aidaMetadata) compareEpochs(firstEpoch uint64, lastEpoch uint64) (uint64, uint64, error) {
	var (
		dbFirst, dbLast uint64
		err             error
	)

	dbFirst, err = m.getFirstEpoch()
	if err != nil {
		if strings.Contains(err.Error(), "leveldb: not found") {
			// block was not found, set to 0
			dbFirst = 0
		} else {
			return 0, 0, err
		}
	}

	if (dbFirst != 0 && dbFirst < firstEpoch) || firstEpoch == 0 {
		firstEpoch = dbFirst
	}

	dbLast, err = m.getLastEpoch()
	if err != nil {
		if strings.Contains(err.Error(), "leveldb: not found") {
			// block was not found, set to 0
			dbLast = 0
		} else {
			return 0, 0, err
		}
	}

	if dbLast > lastEpoch || lastEpoch == 0 {
		lastEpoch = dbLast
	}

	return firstEpoch, lastEpoch, nil
}

// getFirstBlock and return it
func (m *aidaMetadata) getFirstBlock() (uint64, error) {
	firstBlockBytes, err := m.db.Get([]byte(FirstBlockPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get first block; %v", err)
	}

	return bigendian.BytesToUint64(firstBlockBytes), nil
}

// getLastBlock and return it
func (m *aidaMetadata) getLastBlock() (uint64, error) {
	lastBlockBytes, err := m.db.Get([]byte(LastBlockPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get last block; %v", err)
	}

	return bigendian.BytesToUint64(lastBlockBytes), nil
}

// getFirstEpoch and return it
func (m *aidaMetadata) getFirstEpoch() (uint64, error) {
	firstEpochBytes, err := m.db.Get([]byte(FirstEpochPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get first epoch; %v", err)
	}

	return bigendian.BytesToUint64(firstEpochBytes), nil
}

// getLastEpoch and return it
func (m *aidaMetadata) getLastEpoch() (uint64, error) {
	lastEpochBytes, err := m.db.Get([]byte(LastEpochPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get last epoch; %v", err)
	}

	return bigendian.BytesToUint64(lastEpochBytes), nil
}

// getChainID and return it
func (m *aidaMetadata) getChainID() (int, error) {
	chainIDBytes, err := m.db.Get([]byte(ChainIDPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get chain-id; %v", err)
	}

	return int(bigendian.BytesToUint16(chainIDBytes)), nil
}

// getTimestamp and return it
func (m *aidaMetadata) getTimestamp() (uint64, error) {
	byteChainID, err := m.db.Get([]byte(TimestampPrefix))
	if err != nil {
		return 0, fmt.Errorf("cannot get timestamp; %v", err)
	}

	return bigendian.BytesToUint64(byteChainID), nil
}

// getDbType and return it
func (m *aidaMetadata) getDbType() (aidaDbType, error) {
	byteDbType, err := m.db.Get([]byte(TypePrefix))
	if err != nil {
		return noType, fmt.Errorf("cannot get db-type; %v", err)
	}

	return aidaDbType(byteDbType[0]), nil
}

// setFirstBlock in given db
func (m *aidaMetadata) setFirstBlock(firstBlock uint64) error {
	firstBlockBytes := substate.BlockToBytes(firstBlock)

	if err := m.db.Put([]byte(FirstBlockPrefix), firstBlockBytes); err != nil {
		return fmt.Errorf("cannot put first block; %v", err)
	}

	m.log.Info("METADATA: First block saved successfully")

	return nil
}

// setLastBlock in given db
func (m *aidaMetadata) setLastBlock(lastBlock uint64) error {
	lastBlockBytes := substate.BlockToBytes(lastBlock)

	if err := m.db.Put([]byte(LastBlockPrefix), lastBlockBytes); err != nil {
		return fmt.Errorf("cannot put last block; %v", err)
	}

	m.log.Info("METADATA: Last block saved successfully")

	return nil
}

// setFirstEpoch in given db
func (m *aidaMetadata) setFirstEpoch(firstEpoch uint64) error {
	firstEpochBytes := substate.BlockToBytes(firstEpoch)

	if err := m.db.Put([]byte(FirstEpochPrefix), firstEpochBytes); err != nil {
		return fmt.Errorf("cannot put first epoch; %v", err)
	}

	m.log.Info("METADATA: First epoch saved successfully")

	return nil
}

// setLastEpoch in given db
func (m *aidaMetadata) setLastEpoch(lastEpoch uint64) error {
	lastEpochBytes := substate.BlockToBytes(lastEpoch)

	if err := m.db.Put([]byte(LastEpochPrefix), lastEpochBytes); err != nil {
		return fmt.Errorf("cannot put last epoch; %v", err)
	}

	m.log.Info("METADATA: Last epoch saved successfully")

	return nil
}

// setChainID in given db
func (m *aidaMetadata) setChainID(chainID int) error {
	chainIDBytes := bigendian.Uint16ToBytes(uint16(chainID))

	if err := m.db.Put([]byte(ChainIDPrefix), chainIDBytes); err != nil {
		return fmt.Errorf("cannot put chain-id; %v", err)
	}

	m.chainId = chainID

	m.log.Info("METADATA: ChainID saved successfully")

	return nil
}

// setTimestamp in given db
func (m *aidaMetadata) setTimestamp() error {
	createTime := make([]byte, 8)

	binary.BigEndian.PutUint64(createTime, uint64(time.Now().Unix()))
	if err := m.db.Put([]byte(TimestampPrefix), createTime); err != nil {
		return fmt.Errorf("cannot put timestamp into db metadata; %v", err)
	}

	m.log.Info("METADATA: Creation timestamp saved successfully")

	return nil
}

// setDbType in given db
func (m *aidaMetadata) setDbType(dbType aidaDbType) error {
	dbTypeBytes := make([]byte, 1)
	dbTypeBytes[0] = byte(dbType)

	if err := m.db.Put([]byte(TypePrefix), dbTypeBytes); err != nil {
		return fmt.Errorf("cannot put db-type into aida-db; %v", err)
	}

	m.log.Info("METADATA: DB Type saved successfully")

	return nil
}

// setAllMetadata in given db
func (m *aidaMetadata) setAllMetadata(firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int, dbType aidaDbType) error {
	var err error

	if err = m.setFirstBlock(firstBlock); err != nil {
		return err
	}

	if err = m.setLastEpoch(lastBlock); err != nil {
		return err
	}

	if err = m.setFirstEpoch(firstEpoch); err != nil {
		return err
	}

	if err = m.setLastEpoch(lastEpoch); err != nil {
		return err
	}

	if err = m.setChainID(chainID); err != nil {
		return err
	}

	if err = m.setDbType(dbType); err != nil {
		return err
	}

	if err = m.setTimestamp(); err != nil {
		return err
	}
	return nil
}

// findMetadataInSubstate iterates over substate to find first and last block of AidaDb
func (m *aidaMetadata) findMetadataInSubstate(aidaDbPath string) error {

	// todo remove after matejs changes
	m.log.Notice("Iterating through substate to find first and last block and epoch")

	substate.SetSubstateDb(aidaDbPath)
	substate.OpenSubstateDBReadOnly()

	// todo how many workers?
	iter := substate.NewSubstateIterator(0, substate.WorkersFlag.Value)

	defer iter.Release()

	// start with writing first block
	if iter.Next() {
		m.firstBlock = iter.Value().Block
	} else {
		return errors.New("no substate in aida-db")
	}

	m.log.Noticef("Found first block #%v", m.firstBlock)

	var iterLastBlock uint64
	for iter.Next() {
		iterLastBlock = iter.Value().Block

		m.log.Debugf("Block #%v", iterLastBlock)
		if iter.Value().Block%1_000_000 == 0 {
			m.log.Info("Block #%v", iterLastBlock)
		}
	}

	m.lastBlock = iterLastBlock
	m.log.Noticef("Found last block #%v", m.lastBlock)

	return nil
}

// findEpochs for block range in metadata
func (m *aidaMetadata) findEpochs() error {
	var (
		err error
		url string
	)

	if m.chainId == 250 {
		url = RPCMainnet

	} else if m.chainId == 4002 {
		url = RPCTestnet
	}

	firstEpoch, err := findEpochNumber(m.firstBlock, url)
	if err != nil {
		return err
	}

	m.firstEpoch = firstEpoch

	m.log.Noticef("Found first epoch #%v", m.firstEpoch)

	lastEpoch, err := findEpochNumber(m.lastBlock, url)
	if err != nil {
		return err
	}

	m.lastEpoch = lastEpoch

	m.log.Noticef("Found last epoch #%v; patching now continues", m.lastEpoch)

	return nil
}

// checkUpdateMetadata goes through metadata of updated AidaDb and its patch,
// looks if blocks and epoch align and if chainIDs are same for both Dbs
func (m *aidaMetadata) checkUpdateMetadata(isNewDb bool, cfg *utils.Config, patchMD *aidaMetadata) (uint64, uint64, error) {
	var (
		err                    error
		firstBlock, firstEpoch uint64
	)

	if err = patchMD.getMetadata(); err != nil {
		return 0, 0, fmt.Errorf("checkUpdateMetadata patchMD ; %v", err)
	}

	if !isNewDb {
		// if we are updating existing AidaDb and this Db does not have metadata, we go through substate to find
		// blocks and epochs, chainID is set from user via chain-id flag and db type in this case will always be genType
		if err = m.getMetadata(); err != nil {
			// if metadata are not found, but we have an existingDb, we go through substate to find it
			if strings.Contains(err.Error(), "leveldb: not found") {
				MustCloseDB(m.db)

				if err = m.setFreshUpdateMetadata(cfg.ChainID); err != nil {
					return 0, 0, err
				}

			} else {
				return 0, 0, fmt.Errorf("checkUpdateMetadata aida-db ; %v", err)
			}
		}

		// the patch is usable only if its firstBlock is within targetDbs block range
		// and if its last block is bigger than targetDBs last block
		if patchMD.firstBlock > m.lastBlock+1 || patchMD.firstBlock < m.firstBlock || patchMD.lastBlock <= m.lastBlock {
			return 0, 0, fmt.Errorf("metadata blocks does not align; aida-db %v-%v, patch %v-%v", m.firstBlock, m.lastBlock, patchMD.firstBlock, patchMD.lastBlock)
		}

		// if chainIDs doesn't match, we can't patch the DB
		if m.chainId != patchMD.chainId {
			return 0, 0, fmt.Errorf("metadata chain-ids does not match; aida-db: %v, patch: %v", m.chainId, patchMD.chainId)
		}

		// we take first block and epoch from the existing db
		firstBlock = m.firstBlock
		firstEpoch = m.firstEpoch
	} else {
		// if targetDb is a new db, we take first block and epoch from the patch
		firstBlock = patchMD.firstBlock
		firstEpoch = patchMD.firstEpoch
	}

	return firstBlock, firstEpoch, nil
}

// setFreshUpdateMetadata for an existing AidaDb without metadata
func (m *aidaMetadata) setFreshUpdateMetadata(chainID int) error {
	var err error

	if chainID == 0 {
		return fmt.Errorf("since you have AidaDb without metadata you need to specify chain-id (--%v) of your aida-db", utils.ChainIDFlag.Name)
	}

	// ChainID is set by user in
	if err = m.setChainID(chainID); err != nil {
		return err
	}

	if err = m.findEpochs(); err != nil {
		return err
	}

	if err = m.setTimestamp(); err != nil {
		return err
	}

	// updated AidaDb with patches will always be genType
	if err = m.setDbType(genType); err != nil {
		return err
	}

	return nil
}

// findEpochNumber via RPC request getBlockByNumber
func findEpochNumber(blockNumber uint64, url string) (uint64, error) {
	hex := strconv.FormatUint(blockNumber, 16)

	payload := jsonRPCRequest{
		Method:  "ftm_getBlockByNumber",
		Params:  []interface{}{"0x" + hex, false},
		ID:      1,
		JSONRPC: "2.0",
	}

	jsonReq, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("cannot marshal req with first block; %v", err)
	}

	//resp, err := http.Post(RPCMainnet, "application/json", bytes.NewBuffer(jsonReq))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonReq))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	m := make(map[string]interface{})

	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return 0, err
	}

	resultMap, ok := m["result"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpecetd answer: %v", req)
	}

	firstEpochHex, ok := resultMap["epoch"].(string)
	if !ok {
		return 0, fmt.Errorf("cannot find epoch in result: %v", resultMap)
	}

	epoch, ok := math.ParseUint64(firstEpochHex)
	if !ok {
		return 0, fmt.Errorf("cannot parse hex first epoch to uint: %v", firstEpochHex)
	}

	return epoch, nil
}
