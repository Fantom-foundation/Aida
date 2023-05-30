package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/op/go-logging"
)

const (
	TypePrefix       = substate.MetadataPrefix + "ty"
	TimestampPrefix  = substate.MetadataPrefix + "ti"
	FirstBlockPrefix = substate.MetadataPrefix + "fb"
	LastBlockPrefix  = substate.MetadataPrefix + "lb"
	FirstEpochPrefix = substate.MetadataPrefix + "fe"
	LastEpochPrefix  = substate.MetadataPrefix + "le"
	ChainIDPrefix    = substate.MetadataPrefix + "ci"
)

const (
	GenDbType   = "G" // generate
	CloneDbType = "C" // clone
	PatchDbType = "P" // patch

	// merge is determined by what are we merging
	// G + C / C + C / = NOT POSSIBLE
	// G + G = G
	// G + P = G
	// C + P = C
	// P + P = P
)

// MetadataInfo holds any information about AidaDb needed for putting it into the Db
type MetadataInfo struct {
	dbType                aidaDbType
	chainId               int
	firstBlock, lastBlock uint64
	firstEpoch, lastEpoch uint64
}

type metadataFinder struct {
	log                                          *logging.Logger
	mdi                                          *MetadataInfo
	firstBlock, lastBlock, firstEpoch, lastEpoch uint64
}

// processMetadata tries to find data inside give sourceDbs, if not found the ones from config are used
func processMetadata(sourceDbs []ethdb.Database, targetDb ethdb.Database, mdi *MetadataInfo) error {
	var err error

	switch mdi.dbType {
	case genType:
		err = findMetadataGenAndMerge(append(sourceDbs, targetDb), mdi)
		if err != nil {
			return err
		}

		if err = putMetadata(targetDb, mdi); err != nil {
			return err
		}

	case mergeType:
		err = findMetadataGenAndMerge(sourceDbs, mdi)
		if err != nil {
			return err
		}
		return putMetadata(targetDb, mdi)

	case patchType:
		if err = putMetadata(targetDb, mdi); err != nil {
			return err
		}
	case cloneType:
		if err = findMetadataClone(sourceDbs[0], mdi); err != nil {
			return err
		}

		if err = putMetadata(targetDb, mdi); err != nil {
			return err
		}

	default:
		return errors.New("unknown db type")
	}

	return nil
}

// putMetadata decides which put func to call
func putMetadata(targetDb ethdb.Database, mdi *MetadataInfo) error {
	log := logger.NewLogger("INFO", "metadata")

	if err := putBlockMetadata(targetDb, mdi.firstBlock, mdi.lastBlock, log); err != nil {
		return err
	}

	if err := putEpochMetadata(targetDb, mdi.firstEpoch, mdi.lastEpoch, mdi.dbType, log); err != nil {
		return err
	}

	if err := putTimestampMetadata(targetDb); err != nil {
		return err
	}

	if err := putChainIDMetadata(targetDb, mdi.chainId); err != nil {
		return err
	}

	var dbTypeBytes = make([]byte, 1)
	dbTypeBytes[0] = byte(mdi.dbType)

	if err := targetDb.Put([]byte(TypePrefix), dbTypeBytes); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	return nil
}

func putChainIDMetadata(targetDb ethdb.Database, chainID int) error {

	byteChainID := bigendian.Uint16ToBytes(uint16(chainID))

	if err := targetDb.Put([]byte(ChainIDPrefix), byteChainID); err != nil {
		return fmt.Errorf("cannot put chain-id into db metadata; %v", err)
	}

	return nil
}

// findMetadataClone in given sourceDbs - either when Merging or generating AidaDb from substateDb, updatesetDb and deletionDb
func findMetadataClone(sourceDb ethdb.Database, mdi *MetadataInfo) error {

	f := &metadataFinder{
		log: logger.NewLogger("INFO", "Metadata-Finder"),
		mdi: mdi,
	}

	if err := f.findChainID(sourceDb); err != nil {
		return err
	}

	// epochs in clone will not be whole most times, so setting them to 0 is the most logical
	mdi.firstEpoch = 0
	mdi.lastEpoch = 0

	return nil
}

// findMetadataGenAndMerge in given sourceDbs - either when Merging or generating AidaDb from substateDb, updatesetDb and deletionDb
func findMetadataGenAndMerge(sourceDbs []ethdb.Database, mdi *MetadataInfo) error {

	f := &metadataFinder{
		log: logger.NewLogger("INFO", "Metadata-Finder"),
		mdi: mdi,
	}

	// iterate over all dbs to find the first and the last block
	for _, db := range sourceDbs {
		// find what type of db we are merging
		if err := f.findDbType(db); err != nil {
			return err
		}

		if err := f.findChainID(db); err != nil {
			return err
		}

		if err := f.findBlocks(db); err != nil {
			return err
		}

		if err := f.findEpochs(db); err != nil {
			return err
		}
	}

	return nil
}

func (f *metadataFinder) findDbType(db ethdb.Database) error {
	var typStr string

	typByte, err := db.Get([]byte(TypePrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get merge type; %v", err)
		}
		f.log.Warning("cannot find db type")
	} else {
		if err = rlp.DecodeBytes(typByte, &typStr); err != nil {
			return fmt.Errorf("cannot decode merge type; %v", err)
		}
	}

	if typStr == CloneDbType {
		f.mdi.dbType = cloneType
	}

	return nil
}

func (f *metadataFinder) findBlocks(db ethdb.Database) error {
	var first, last uint64

	firstBlockBytes, err := db.Get([]byte(FirstBlockPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get first block ; %v", err)
		}
		return nil
	}

	first = bigendian.BytesToUint64(firstBlockBytes)

	if first < f.mdi.firstBlock {
		f.mdi.firstBlock = first
	}

	lastBlockBytes, err := db.Get([]byte(LastBlockPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get last block; %v", err)
		}
		return nil
	}

	last = bigendian.BytesToUint64(lastBlockBytes)

	if last > f.mdi.lastBlock {
		f.mdi.lastBlock = last
	}

	return nil
}

func (f *metadataFinder) findEpochs(db ethdb.Database) error {
	var first, last uint64

	firstEpochBytes, err := db.Get([]byte(FirstEpochPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get first epoch ; %v", err)
		}
		return nil
	}

	first = bigendian.BytesToUint64(firstEpochBytes)

	if first < f.mdi.firstEpoch {
		f.mdi.firstEpoch = first
	}

	lastEpochBytes, err := db.Get([]byte(LastEpochPrefix))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cannot get last epoch; %v", err)
		}
		return nil
	}

	last = bigendian.BytesToUint64(lastEpochBytes)

	if last > f.mdi.lastEpoch {
		f.mdi.lastEpoch = last
	}

	return nil
}

func (f *metadataFinder) findChainID(db ethdb.Database) error {
	byteChainID, err := db.Get([]byte(ChainIDPrefix))
	if err != nil {
		return fmt.Errorf("cannot get chain-id from aida-db; %v", err)
	}

	f.mdi.chainId = int(bigendian.BytesToUint16(byteChainID))

	return nil
}

// putBlockMetadata into AidaDb
func putBlockMetadata(targetDb ethdb.Database, firstBlock, lastBlock uint64, log *logging.Logger) error {
	if firstBlock == 0 {
		log.Warning("given first block is 0 - saving to metadata anyway")
	}

	firstBlockBytes := substate.BlockToBytes(firstBlock)
	if err := targetDb.Put([]byte(FirstBlockPrefix), firstBlockBytes); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	if lastBlock == 0 {
		log.Warning("given last block is 0 - saving to metadata anyway")
	}

	lastBlockBytes := substate.BlockToBytes(lastBlock)
	if err := targetDb.Put([]byte(LastBlockPrefix), lastBlockBytes); err != nil {
		return fmt.Errorf("cannot put last block number into db metadata; %v", err)
	}

	return nil
}

// putEpochMetadata into AidaDb
func putEpochMetadata(targetDb ethdb.Database, firstEpoch, lastEpoch uint64, dbType aidaDbType, log *logging.Logger) error {

	if firstEpoch == 0 {
		log.Warning("given first epoch is 0 - saving to metadata anyway")
	}

	firstEpochBytes := substate.BlockToBytes(firstEpoch)
	if err := targetDb.Put([]byte(FirstEpochPrefix), firstEpochBytes); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	if lastEpoch == 0 {
		log.Warning("given last epoch is 0 - saving to metadata anyway")
	}

	// if db is type of clone, epochs are set to 0
	if dbType != cloneType {
		lastEpoch -= 1
	}

	lastEpochBytes := substate.BlockToBytes(lastEpoch)
	if err := targetDb.Put([]byte(LastEpochPrefix), lastEpochBytes); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	return nil
}

// putTimestampMetadata into AidaDb
func putTimestampMetadata(targetDb ethdb.Database) error {
	createTime := make([]byte, 8)

	binary.BigEndian.PutUint64(createTime, uint64(time.Now().Unix()))
	if err := targetDb.Put([]byte(TimestampPrefix), createTime); err != nil {
		return fmt.Errorf("cannot put timestamp into db metadata; %v", err)
	}

	return nil
}
