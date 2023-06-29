package utils

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"github.com/syndtr/goleveldb/leveldb"
)

type AidaDbType byte

const (
	NoType AidaDbType = iota
	GenType
	PatchType
	CloneType
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
// genType + CloneType / CloneType + CloneType / = NOT POSSIBLE
// genType + genType = genType
// genType + PatchType = genType
// CloneType + PatchType = CloneType
// PatchType + PatchType = PatchType

// AidaDbMetadata holds any information about AidaDb needed for putting it into the Db
type AidaDbMetadata struct {
	Db                    ethdb.Database
	log                   *logging.Logger
	FirstBlock, LastBlock uint64
	FirstEpoch, LastEpoch uint64
	ChainId               int
	DbType                AidaDbType
	timestamp             uint64
}

// todo we need to check block alignment and chainID match before any merging

// NewAidaDbMetadata creates new instance of AidaDbMetadata
func NewAidaDbMetadata(db ethdb.Database, logLevel string) *AidaDbMetadata {
	return &AidaDbMetadata{
		Db:  db,
		log: logger.NewLogger(logLevel, "aida-metadata"),
	}
}

// ProcessPatchLikeMetadata decides whether patch is new or not. If so the DbType is Set to GenType, otherwise its PatchType.
// Then it inserts all given metadata
func ProcessPatchLikeMetadata(aidaDb ethdb.Database, logLevel string, firstBlock, lastBlock, firstEpoch, lastEpoch uint64, chainID int, isNew bool) error {
	var (
		dbType AidaDbType
		err    error
	)

	// if this is brand-new patch, it should be treated as a gen type db
	if isNew {
		dbType = GenType
	} else {
		dbType = PatchType
	}

	md := NewAidaDbMetadata(aidaDb, logLevel)

	if err = md.SetFirstBlock(firstBlock); err != nil {
		return err
	}
	if err = md.SetLastBlock(lastBlock); err != nil {
		return err
	}

	if err = md.SetFirstEpoch(firstEpoch); err != nil {
		return err
	}
	if err = md.SetLastEpoch(lastEpoch); err != nil {
		return err
	}

	if err = md.SetChainID(chainID); err != nil {
		return err
	}

	if err = md.SetDbType(dbType); err != nil {
		return err
	}

	if err = md.SetTimestamp(); err != nil {
		return err
	}

	md.log.Notice("Metadata added successfully")

	return nil
}

// ProcessCloneLikeMetadata inserts every metadata from sourceDb, only epochs are excluded.
// We can't be certain if given epoch is whole
func ProcessCloneLikeMetadata(aidaDb ethdb.Database, logLevel string, firstBlock, lastBlock uint64, chainID int) error {
	var err error

	md := NewAidaDbMetadata(aidaDb, logLevel)

	firstBlock, lastBlock = md.compareBlocks(firstBlock, lastBlock)

	if err = md.SetFirstBlock(firstBlock); err != nil {
		return err
	}
	if err = md.SetLastBlock(lastBlock); err != nil {
		return err
	}

	if err = md.SetChainID(chainID); err != nil {
		return err
	}

	if err = md.findEpochs(); err != nil {
		return err
	}

	if err = md.SetFirstEpoch(md.FirstEpoch); err != nil {
		return err
	}

	if err = md.SetLastEpoch(md.LastEpoch); err != nil {
		return err
	}

	if err = md.SetDbType(CloneType); err != nil {
		return err
	}

	if err = md.SetTimestamp(); err != nil {
		return err
	}

	md.log.Notice("Metadata added successfully")
	return nil
}

func ProcessGenLikeMetadata(pathToAidaDb string, firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int, logLevel string) error {
	aidaDb, err := rawdb.NewLevelDBDatabase(pathToAidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("metadata cannot open AidaDb; %v", err)
	}

	defer aidaDb.Close()

	md := NewAidaDbMetadata(aidaDb, logLevel)
	return md.genMetadata(firstBlock, lastBlock, firstEpoch, lastEpoch, chainID)
}

// genMetadata inserts metadata into newly generated AidaDb.
// If generate is used onto an existing AidaDb it updates last block, last epoch and timestamp.
func (md *AidaDbMetadata) genMetadata(firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int) error {
	var err error

	firstBlock, lastBlock = md.compareBlocks(firstBlock, lastBlock)

	if err = md.SetFirstBlock(firstBlock); err != nil {
		return err
	}
	if err = md.SetLastBlock(lastBlock); err != nil {
		return err
	}

	firstEpoch, lastEpoch = md.compareEpochs(firstEpoch, lastEpoch)

	if err = md.SetFirstEpoch(firstEpoch); err != nil {
		return err
	}
	if err = md.SetLastEpoch(lastEpoch); err != nil {
		return err
	}

	if err = md.SetChainID(chainID); err != nil {
		return err
	}

	if err = md.SetDbType(GenType); err != nil {
		return err
	}

	if err = md.SetTimestamp(); err != nil {
		return err
	}

	return nil
}

// ProcessMergeMetadata decides the type according to the types of merged Dbs and inserts every metadata
func ProcessMergeMetadata(cfg *Config, aidaDb ethdb.Database, sourceDbs []ethdb.Database, paths []string) (*AidaDbMetadata, error) {
	var (
		err error
	)

	targetMD := NewAidaDbMetadata(aidaDb, cfg.LogLevel)

	for i, db := range sourceDbs {
		md := NewAidaDbMetadata(db, cfg.LogLevel)
		md.GetMetadata()

		// todo do we need to check whether blocks align?

		// Get chainID of first source db
		if targetMD.ChainId == 0 {
			targetMD.ChainId = md.ChainId
		}

		// if chain ids doesn't match, we should not be merging
		if md.ChainId != targetMD.ChainId {
			md.log.Critical("ChainIDs in Dbs metadata does not match!")
		}

		// if db had no metadata we will look for blocks in substate
		if md.FirstBlock == 0 {
			// we need to close db before opening substate
			if err = db.Close(); err != nil {
				return nil, fmt.Errorf("cannot close db; %v", err)
			}

			md.FirstBlock, md.LastBlock, err = FindBlockRangeInSubstate(paths[i])
			if err != nil {
				return nil, fmt.Errorf("cannot find blocks in substate; %v", err)
			}

			// reopen db
			md.Db, err = rawdb.NewLevelDBDatabase(paths[i], 1024, 100, "profiling", true)
			if err != nil {
				return nil, fmt.Errorf("cannot open aida-db; %v", err)
			}
		}

		if md.FirstBlock < targetMD.FirstBlock || targetMD.FirstBlock == 0 {
			targetMD.FirstBlock = md.FirstBlock
		}

		if md.LastBlock > targetMD.LastBlock || targetMD.LastBlock == 0 {
			targetMD.LastBlock = md.LastBlock
		}

		// set first
		if targetMD.DbType == NoType {
			targetMD.DbType = md.DbType
			continue
		}

		if targetMD.DbType == GenType && (md.DbType == PatchType || md.DbType == GenType) {
			targetMD.DbType = GenType
			continue
		}

		if targetMD.DbType == PatchType {
			switch md.DbType {
			case GenType:
				targetMD.DbType = GenType
				continue
			case PatchType:
				targetMD.DbType = PatchType
				continue
			case CloneType:
				targetMD.DbType = CloneType
				continue
			}
		}

		if targetMD.DbType == CloneType && md.DbType == PatchType {
			targetMD.DbType = CloneType
			continue
		}

		return nil, fmt.Errorf("cannot merge %v with %v", targetMD.getVerboseDbType(), md.getVerboseDbType())
	}

	if err = targetMD.findEpochs(); err != nil {
		return nil, err
	}

	if targetMD.ChainId == 0 {
		targetMD.log.Warningf("your dbs does not have chain-id, Setting value from config (%v)", cfg.ChainID)
		targetMD.ChainId = cfg.ChainID
	}

	return targetMD, nil
}

// GetMetadata from given Db and save it
func (md *AidaDbMetadata) GetMetadata() {
	md.FirstBlock = md.GetFirstBlock()

	md.LastBlock = md.GetLastBlock()

	md.FirstEpoch = md.GetFirstEpoch()

	md.LastEpoch = md.GetLastEpoch()

	md.DbType = md.GetDbType()

	md.timestamp = md.GetTimestamp()

	md.ChainId = md.GetChainID()
}

// compareBlocks from given Db and return them
func (md *AidaDbMetadata) compareBlocks(firstBlock uint64, lastBlock uint64) (uint64, uint64) {
	var (
		dbFirst, dbLast uint64
	)

	dbFirst = md.GetFirstBlock()
	if (dbFirst != 0 && dbFirst < firstBlock) || firstBlock == 0 {
		firstBlock = dbFirst
	}

	dbLast = md.GetLastBlock()

	if dbLast > lastBlock || lastBlock == 0 {
		lastBlock = dbLast
	}

	return firstBlock, lastBlock
}

// compareEpochs from given Db and return them
func (md *AidaDbMetadata) compareEpochs(firstEpoch uint64, lastEpoch uint64) (uint64, uint64) {
	var (
		dbFirst, dbLast uint64
	)

	dbFirst = md.GetFirstEpoch()
	if (dbFirst != 0 && dbFirst < firstEpoch) || firstEpoch == 0 {
		firstEpoch = dbFirst
	}

	dbLast = md.GetLastEpoch()
	if dbLast > lastEpoch || lastEpoch == 0 {
		lastEpoch = dbLast
	}

	return firstEpoch, lastEpoch
}

// GetFirstBlock and return it
func (md *AidaDbMetadata) GetFirstBlock() uint64 {
	firstBlockBytes, err := md.Db.Get([]byte(FirstBlockPrefix))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 0
		}
		md.log.Criticalf("cannot get first block from metadata; %v", err)
		return 0
	}

	return bigendian.BytesToUint64(firstBlockBytes)
}

// GetLastBlock and return it
func (md *AidaDbMetadata) GetLastBlock() uint64 {
	lastBlockBytes, err := md.Db.Get([]byte(LastBlockPrefix))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 0
		}
		md.log.Criticalf("cannot get last block from metadata; %v", err)
		return 0
	}

	return bigendian.BytesToUint64(lastBlockBytes)
}

// GetFirstEpoch and return it
func (md *AidaDbMetadata) GetFirstEpoch() uint64 {
	firstEpochBytes, err := md.Db.Get([]byte(FirstEpochPrefix))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 0
		}
		md.log.Criticalf("cannot get first epoch from metadata; %v", err)
		return 0
	}

	return bigendian.BytesToUint64(firstEpochBytes)
}

// GetLastEpoch and return it
func (md *AidaDbMetadata) GetLastEpoch() uint64 {
	lastEpochBytes, err := md.Db.Get([]byte(LastEpochPrefix))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 0
		}
		md.log.Criticalf("cannot get last epoch from metadata; %v", err)
		return 0
	}

	return bigendian.BytesToUint64(lastEpochBytes)
}

// GetChainID and return it
func (md *AidaDbMetadata) GetChainID() int {
	chainIDBytes, err := md.Db.Get([]byte(ChainIDPrefix))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 0
		}
		md.log.Criticalf("cannot get chain id from metadata; %v", err)
		return 0
	}

	return int(bigendian.BytesToUint16(chainIDBytes))
}

// GetTimestamp and return it
func (md *AidaDbMetadata) GetTimestamp() uint64 {
	byteTimestamp, err := md.Db.Get([]byte(TimestampPrefix))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 0
		}
		md.log.Criticalf("cannot get timestamp from metadata; %v", err)
		return 0
	}

	return bigendian.BytesToUint64(byteTimestamp)
}

// GetDbType and return it
func (md *AidaDbMetadata) GetDbType() AidaDbType {
	byteDbType, err := md.Db.Get([]byte(TypePrefix))
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return NoType
		}
		md.log.Criticalf("cannot get db type from metadata; %v", err)
		return 0
	}

	return AidaDbType(byteDbType[0])
}

// SetFirstBlock in given Db
func (md *AidaDbMetadata) SetFirstBlock(firstBlock uint64) error {
	firstBlockBytes := substate.BlockToBytes(firstBlock)

	if err := md.Db.Put([]byte(FirstBlockPrefix), firstBlockBytes); err != nil {
		return fmt.Errorf("cannot put first block; %v", err)
	}

	md.FirstBlock = firstBlock

	md.log.Info("METADATA: First block saved successfully")

	return nil
}

// SetLastBlock in given Db
func (md *AidaDbMetadata) SetLastBlock(lastBlock uint64) error {
	lastBlockBytes := substate.BlockToBytes(lastBlock)

	if err := md.Db.Put([]byte(LastBlockPrefix), lastBlockBytes); err != nil {
		return fmt.Errorf("cannot put last block; %v", err)
	}

	md.LastBlock = lastBlock

	md.log.Info("METADATA: Last block saved successfully")

	return nil
}

// SetFirstEpoch in given Db
func (md *AidaDbMetadata) SetFirstEpoch(firstEpoch uint64) error {
	firstEpochBytes := substate.BlockToBytes(firstEpoch)

	if err := md.Db.Put([]byte(FirstEpochPrefix), firstEpochBytes); err != nil {
		return fmt.Errorf("cannot put first epoch; %v", err)
	}

	md.log.Info("METADATA: First epoch saved successfully")

	return nil
}

// SetLastEpoch in given Db
func (md *AidaDbMetadata) SetLastEpoch(lastEpoch uint64) error {
	lastEpochBytes := substate.BlockToBytes(lastEpoch)

	if err := md.Db.Put([]byte(LastEpochPrefix), lastEpochBytes); err != nil {
		return fmt.Errorf("cannot put last epoch; %v", err)
	}

	md.log.Info("METADATA: Last epoch saved successfully")

	return nil
}

// SetChainID in given Db
func (md *AidaDbMetadata) SetChainID(chainID int) error {
	chainIDBytes := bigendian.Uint16ToBytes(uint16(chainID))

	if err := md.Db.Put([]byte(ChainIDPrefix), chainIDBytes); err != nil {
		return fmt.Errorf("cannot put chain-id; %v", err)
	}

	md.ChainId = chainID

	md.log.Info("METADATA: ChainID saved successfully")

	return nil
}

// SetTimestamp in given Db
func (md *AidaDbMetadata) SetTimestamp() error {
	createTime := make([]byte, 8)

	binary.BigEndian.PutUint64(createTime, uint64(time.Now().Unix()))
	if err := md.Db.Put([]byte(TimestampPrefix), createTime); err != nil {
		return fmt.Errorf("cannot put timestamp into db metadata; %v", err)
	}

	md.log.Info("METADATA: Creation timestamp saved successfully")

	return nil
}

// SetDbType in given Db
func (md *AidaDbMetadata) SetDbType(dbType AidaDbType) error {
	dbTypeBytes := make([]byte, 1)
	dbTypeBytes[0] = byte(dbType)

	if err := md.Db.Put([]byte(TypePrefix), dbTypeBytes); err != nil {
		return fmt.Errorf("cannot put db-type into aida-db; %v", err)
	}

	md.log.Info("METADATA: DB Type saved successfully")

	return nil
}

// SetAllMetadata in given Db
func (md *AidaDbMetadata) SetAllMetadata(firstBlock uint64, lastBlock uint64, firstEpoch uint64, lastEpoch uint64, chainID int, dbType AidaDbType) error {
	var err error

	if err = md.SetFirstBlock(firstBlock); err != nil {
		return err
	}

	if err = md.SetLastBlock(lastBlock); err != nil {
		return err
	}

	if err = md.SetFirstEpoch(firstEpoch); err != nil {
		return err
	}

	if err = md.SetLastEpoch(lastEpoch); err != nil {
		return err
	}

	if err = md.SetChainID(chainID); err != nil {
		return err
	}

	if err = md.SetDbType(dbType); err != nil {
		return err
	}

	if err = md.SetTimestamp(); err != nil {
		return err
	}
	return nil
}

// findEpochs for block range in metadata
func (md *AidaDbMetadata) findEpochs() error {
	var (
		err                            error
		testnet                        bool
		firstEpochMinus, lastEpochPlus uint64
	)

	if md.ChainId == 250 {
		testnet = false
	} else if md.ChainId == 4002 {
		testnet = true
	} else {
		return fmt.Errorf("unknown chain-id %v", md.ChainId)
	}

	md.FirstEpoch, err = findEpochNumber(md.FirstBlock, testnet)
	if err != nil {
		return err
	}

	// we need to check if block is really first block of an epoch
	firstEpochMinus, err = findEpochNumber(md.FirstBlock-1, testnet)
	if err != nil {
		return err
	}

	if firstEpochMinus >= md.FirstEpoch {
		md.log.Warningf("first block of db is not beginning of an epoch; setting first epoch to 0")
		md.FirstEpoch = 0
	} else {
		md.log.Noticef("Found first epoch #%v", md.FirstEpoch)
	}

	md.LastEpoch, err = findEpochNumber(md.LastBlock, testnet)
	if err != nil {
		return err
	}

	// we need to check if block is really last block of an epoch
	lastEpochPlus, err = findEpochNumber(md.LastBlock+1, testnet)
	if err != nil {
		return err
	}

	if lastEpochPlus <= md.LastEpoch {
		md.log.Warningf("last block block of db is not end of an epoch; setting last epoch to 0")
		md.LastEpoch = 0
	} else {
		md.log.Noticef("Found last epoch #%v", md.LastEpoch)
	}

	return nil
}

// CheckUpdateMetadata goes through metadata of updated AidaDb and its patch,
// looks if blocks and epoch align and if chainIDs are same for both Dbs
func (md *AidaDbMetadata) CheckUpdateMetadata(cfg *Config, patchMD *AidaDbMetadata) (uint64, uint64, error) {
	var (
		err                    error
		firstBlock, firstEpoch uint64
	)

	patchMD.GetMetadata()

	// if we are updating existing AidaDb and this Db does not have metadata, we go through substate to find
	// blocks and epochs, chainID is Set from user via chain-id flag and db type in this case will always be genType
	md.GetMetadata()
	if md.FirstBlock == 0 {
		if err = md.SetFreshUpdateMetadata(cfg.ChainID); err != nil {
			return 0, 0, err
		}
	}

	// the patch is usable only if its FirstBlock is within tarGetDbs block range
	// and if its last block is bigger than tarGetDBs last block
	if patchMD.FirstBlock > md.LastBlock+1 || patchMD.FirstBlock < md.FirstBlock || patchMD.LastBlock <= md.LastBlock {
		return 0, 0, fmt.Errorf("metadata blocks does not align; aida-db %v-%v, patch %v-%v", md.FirstBlock, md.LastBlock, patchMD.FirstBlock, patchMD.LastBlock)
	}

	// if chainIDs doesn't match, we can't patch the DB
	if md.ChainId != patchMD.ChainId {
		return 0, 0, fmt.Errorf("metadata chain-ids does not match; aida-db: %v, patch: %v", md.ChainId, patchMD.ChainId)
	}

	// we take first block and epoch from the existing db
	firstBlock = md.FirstBlock
	firstEpoch = md.FirstEpoch

	return firstBlock, firstEpoch, nil
}

// SetFreshUpdateMetadata for an existing AidaDb without metadata
func (md *AidaDbMetadata) SetFreshUpdateMetadata(chainID int) error {
	var err error

	if chainID == 0 {
		return fmt.Errorf("since you have AidaDb without metadata you need to specify chain-id (--%v) of your aida-db", ChainIDFlag.Name)
	}

	// ChainID is Set by user in
	if err = md.SetChainID(chainID); err != nil {
		return err
	}

	if err = md.findEpochs(); err != nil {
		return err
	}

	if err = md.SetTimestamp(); err != nil {
		return err
	}

	// updated AidaDb with patches will always be genType
	if err = md.SetDbType(GenType); err != nil {
		return err
	}

	return nil
}

func (md *AidaDbMetadata) SetBlockRange(firstBlock uint64, lastBlock uint64) error {
	var err error

	if err = md.SetFirstBlock(firstBlock); err != nil {
		return err
	}
	if err = md.SetLastBlock(lastBlock); err != nil {
		return err
	}

	return nil
}

func (md *AidaDbMetadata) DeleteMetadata() {
	var err error

	if err = md.Db.Delete([]byte(ChainIDPrefix)); err != nil {
		md.log.Criticalf("cannot delete chain-id; %v", err)
	} else {
		md.log.Debugf("ChainID deleted successfully")
	}

	if err = md.Db.Delete([]byte(FirstBlockPrefix)); err != nil {
		md.log.Criticalf("cannot delete first block; %v", err)
	} else {
		md.log.Debugf("First block deleted successfully")
	}

	if err = md.Db.Delete([]byte(LastBlockPrefix)); err != nil {
		md.log.Criticalf("cannot delete last block; %v", err)
	} else {
		md.log.Debugf("Last block deleted successfully")
	}

	if err = md.Db.Delete([]byte(FirstEpochPrefix)); err != nil {
		md.log.Criticalf("cannot delete first epoch; %v", err)
	} else {
		md.log.Debugf("First epoch deleted successfully")
	}

	if err = md.Db.Delete([]byte(LastEpochPrefix)); err != nil {
		md.log.Criticalf("cannot delete last epoch; %v", err)
	} else {
		md.log.Debugf("Last epoch deleted successfully")
	}

	if err = md.Db.Delete([]byte(TypePrefix)); err != nil {
		md.log.Criticalf("cannot delete db type; %v", err)
	} else {
		md.log.Debugf("Timestamp deleted successfully")
	}

	if err = md.Db.Delete([]byte(TimestampPrefix)); err != nil {
		md.log.Criticalf("cannot delete creation timestamp; %v", err)
	} else {
		md.log.Debugf("Timestamp deleted successfully")
	}
}

// UpdateMetadataInOldAidaDb Sets metadata necessary for update in old aida-db, which doesn't have any metadata
func (md *AidaDbMetadata) UpdateMetadataInOldAidaDb(chainId int, firstAidaDbBlock uint64, lastAidaDbBlock uint64) error {
	var err error

	// Set chainid if it doesn't exist
	inCID := md.GetChainID()
	if inCID == 0 {
		err = md.SetChainID(chainId)
		if err != nil {
			return err
		}
	}

	// Set first block if it doesn't exist
	inFB := md.GetFirstBlock()
	if inFB == 0 {
		err = md.SetFirstBlock(firstAidaDbBlock)
		if err != nil {
			return err
		}
	}

	// Set last block if it doesn't exist
	inLB := md.GetLastBlock()
	if inLB == 0 {
		err = md.SetLastBlock(lastAidaDbBlock)
		if err != nil {
			return err
		}
	}

	return nil
}

func (md *AidaDbMetadata) getVerboseDbType() string {
	switch md.DbType {
	case GenType:
		return "Generate"
	case CloneType:
		return "Clone"
	case PatchType:
		return "Patch"
	case NoType:
		return "NoType"

	default:
		return "unknown db type"
	}
}

// findEpochNumber via RPC request GetBlockByNumber
func findEpochNumber(blockNumber uint64, testnet bool) (uint64, error) {
	hex := strconv.FormatUint(blockNumber, 16)

	payload := JsonRPCRequest{
		Method:  "ftm_getBlockByNumber",
		Params:  []interface{}{"0x" + hex, false},
		ID:      1,
		JSONRPC: "2.0",
	}

	m, err := SendRPCRequest(payload, testnet)
	if err != nil {
		return 0, err
	}

	resultMap, ok := m["result"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpecetd answer: %v", m)
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

// FindBlockRangeInSubstate if AidaDb does not yet have metadata
func FindBlockRangeInSubstate(pathToAidaDb string) (uint64, uint64, error) {
	substate.SetSubstateDb(pathToAidaDb)
	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()

	firstSubstate := substate.GetFirstSubstate()
	if firstSubstate == nil {
		return 0, 0, errors.New("unable to Get first substate from AidaDb")
	}
	firstBlock := firstSubstate.Env.Number

	lastSubstate, err := substate.GetLastSubstate()
	if err != nil {
		return 0, 0, err
	}
	if lastSubstate == nil {
		return 0, 0, errors.New("unable to Get last substate from AidaDb")
	}
	lastBlock := lastSubstate.Env.Number

	return firstBlock, lastBlock, nil
}
