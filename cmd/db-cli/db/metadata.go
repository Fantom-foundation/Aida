package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
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
)

const (
	GenDbType   = "G" // generate
	CloneDbType = "C" // clone
	PatchDbType = "P" // patch

	// merge is determined by what are we merging
	// G + C / C + C / P + P = NOT POSSIBLE
	// G + G = G
	// G + P = G
	// C + P = C
)

type metadataInfo struct {
	dbType                aidaDbType
	firstBlock, lastBlock uint64
	firstEpoch, lastEpoch string
}

// processMetadata tries to find data inside give sourceDbs, if not found the ones from config are used
func processMetadata(sourceDbs []ethdb.Database, targetDb ethdb.Database, mdi metadataInfo) error {
	switch mdi.dbType {
	case genType:
		if err := putMetadata(targetDb, mdi); err != nil {
			return err
		}

	case mergeType:
		if err := processMergeTypeMetadata(sourceDbs, targetDb, mdi); err != nil {
			return err
		}

	case patchType:
		if err := putMetadata(targetDb, mdi); err != nil {
			return err
		}

	default:
		return errors.New("unknown db type")
	}

	return nil
}

func processMergeTypeMetadata(sourceDbs []ethdb.Database, targetDb ethdb.Database, mdi metadataInfo) error {
	var err error

	mdi.firstBlock, mdi.lastBlock, mdi.dbType, err = findMetadata(sourceDbs)
	if err != nil {
		return err
	}

	return putMetadata(targetDb, mdi)
}

func putMetadata(targetDb ethdb.Database, mdi metadataInfo) error {
	log := logger.NewLogger("INFO", "metadata")

	if err := putBlockMetadata(targetDb, mdi.lastBlock, mdi.firstBlock, log); err != nil {
		return err
	}

	if err := putEpochMetadata(targetDb, mdi.firstEpoch, mdi.lastEpoch, log); err != nil {
		return err
	}

	if err := putTimestampMetadata(targetDb); err != nil {
		return err
	}

	if err := targetDb.Put([]byte(TypePrefix), []byte(GenDbType)); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	return nil
}

func findMetadata(sourceDbs []ethdb.Database) (uint64, uint64, aidaDbType, error) {
	var (
		first, last, totalFirst, totalLast uint64
		mergeTypes                         []string
		log                                = logger.NewLogger("INFO", "process-metadata")
		typStr                             string
	)

	// iterate over all dbs to find the first and the last block
	for i, db := range sourceDbs {
		// find what type of db we are merging
		typByte, err := db.Get([]byte(TypePrefix))
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return 0, 0, 0, fmt.Errorf("cannot get merge type from %v. db; %v", i, err)
			}
			log.Warningf("cannot find db type for %v. db", i)
		} else {
			if err := rlp.DecodeBytes(typByte, &typStr); err != nil {
				return 0, 0, 0, fmt.Errorf("cannot decode merge type %v. db; %v", i, err)
			}

			mergeTypes = append(mergeTypes)
		}

		firstBlockBytes, err := db.Get([]byte(FirstBlockPrefix))
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return 0, 0, 0, fmt.Errorf("cannot get first block from %v. db; %v", i, err)
			}
			continue
		}

		first = bigendian.BytesToUint64(firstBlockBytes)

		if first < totalFirst {
			totalFirst = first
		}

		lastBlockBytes, err := db.Get([]byte(LastBlockPrefix))
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return 0, 0, 0, fmt.Errorf("cannot get last block from %v. db; %v", i, err)
			}
			continue
		}

		last = bigendian.BytesToUint64(lastBlockBytes)

		if last > totalLast {
			totalLast = last
		}
	}

	for _, mt := range mergeTypes {

		// if any of db is clone type return immediately
		if mt == CloneDbType {
			return totalFirst, totalLast, cloneType, nil
		}
	}

	return totalFirst, totalLast, genType, nil
}

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

func putEpochMetadata(targetDb ethdb.Database, firstEpoch, lastEpoch string, log *logging.Logger) error {
	first, err := strconv.ParseUint(firstEpoch, 10, 64)
	if err != nil {
		return fmt.Errorf("parse first epoch; %v", err)
	}

	if first == 0 {
		log.Warning("given first epoch is 0 - saving to metadata anyway")
	}

	firstEpochBytes := substate.BlockToBytes(first)
	if err := targetDb.Put([]byte(FirstEpochPrefix), firstEpochBytes); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	last, err := strconv.ParseUint(lastEpoch, 10, 64)
	if err != nil {
		return fmt.Errorf("parse first epoch; %v", err)
	}

	if last == 0 {
		log.Warning("given last epoch is 0 - saving to metadata anyway")
	}

	lastEpochBytes := substate.BlockToBytes(last)
	if err := targetDb.Put([]byte(LastEpochPrefix), lastEpochBytes); err != nil {
		return fmt.Errorf("cannot put first block number into db metadata; %v", err)
	}

	return nil
}

func putTimestampMetadata(targetDb ethdb.Database) error {
	createTime := make([]byte, 8)

	binary.BigEndian.PutUint64(createTime, uint64(time.Now().UTC().Second()))
	if err := targetDb.Put([]byte(TimestampPrefix), createTime); err != nil {
		return fmt.Errorf("cannot put timestamp into db metadata; %v", err)
	}

	return nil
}
