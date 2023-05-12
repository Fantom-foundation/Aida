package db

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// rawEntry representation of item in database
type rawEntry struct {
	Key   []byte
	Value []byte
}

var dbItemChanSize = 100_000

// CloneCommand enables creation of aida-db copy or subset
var CloneCommand = cli.Command{
	Action: clone,
	Name:   "clone",
	Usage:  "clone can create aida-db copy or subset",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.TargetDbFlag,
		&utils.CompactDbFlag,
		&utils.LogLevelFlag,
	},
	Description: `
Creates clone of aida-db for desired block range
`,
}

// clone creates aida-db copy or subset
func clone(ctx *cli.Context) error {
	//	N, first block
	//	M, last block
	//	cn, last updateset block before N
	//	cm, last updateset block before M
	//
	//	deletion db: 1 to M (whole database is transferred instead since it is small)
	//	update db: 1 to cm
	//	substate : cn to M

	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	log := utils.NewLogger(cfg.LogLevel, "DB Clone")

	aidaDb, targetDb, err := openCloneDatabases(cfg)
	if err != nil {
		return err
	}

	if cfg.CompactDb {
		log.Noticef("Starting compaction")
		err = targetDb.Compact(nil, nil)
		if err != nil {
			return err
		}
	}

	// open writing channel
	writerChan, errChan := writeDataAsync(targetDb)

	// write all contract codes
	err = write(writerChan, errChan, aidaDb.NewIterator([]byte(substate.Stage1CodePrefix), nil), nil)
	if err != nil {
		return err
	}

	// write all destroyed accounts
	err = write(writerChan, errChan, aidaDb.NewIterator([]byte(substate.DestroyedAccountPrefix), nil), nil)
	if err != nil {
		return err
	}

	// write update sets until cfg.Last
	var lastUpdateBeforeRange uint64
	lastUpdateBeforeRange, err = writeUpdateSet(cfg, writerChan, errChan, aidaDb)
	if err != nil {
		return err
	}

	// write substates from last updateset before cfg.First until cfg.Last
	err = writeSubstates(cfg, writerChan, errChan, aidaDb, lastUpdateBeforeRange)
	if err != nil {
		return err
	}

	// close aida database
	MustCloseDB(aidaDb)
	// close target database
	MustCloseDB(targetDb)

	return nil

}

// writeSubstates write substates from last updateset before cfg.First until cfg.Last
func writeSubstates(cfg *utils.Config, writerChan chan rawEntry, errChan chan error, aidaDb ethdb.Database, lastUpdateBeforeRange uint64) error {
	endCond := func(key []byte) (bool, error) {
		block, _, err := substate.DecodeStage1SubstateKey(key)
		if err != nil {
			return false, err
		}
		if block > cfg.Last {
			return true, nil
		}
		return false, nil
	}
	// generating substates right after previous updateset before our interval
	return write(writerChan, errChan, aidaDb.NewIterator([]byte(substate.Stage1SubstatePrefix), substate.BlockToBytes(lastUpdateBeforeRange+1)), endCond)
}

// writeUpdateSet write update sets until cfg.Last
func writeUpdateSet(cfg *utils.Config, writerChan chan rawEntry, errChan chan error, aidaDb ethdb.Database) (uint64, error) {
	// labeling last updateset before interval - need to export substates for that range as well
	var lastUpdateBeforeRange uint64
	endCond := func(key []byte) (bool, error) {
		block, err := substate.DecodeUpdateSetKey(key)
		if err != nil {
			return false, err
		}
		if block > cfg.Last {
			return true, nil
		}
		if block < cfg.First {
			lastUpdateBeforeRange = block
		}
		return false, nil
	}
	// check if updateset contained at least one set (first set with worldstate), then aida-db must be corrupted
	if lastUpdateBeforeRange == 0 {
		return 0, fmt.Errorf("updateset didn't contain any records; unable to create aida-db without initial world-state")
	}

	return lastUpdateBeforeRange, write(writerChan, errChan, aidaDb.NewIterator([]byte(substate.SubstateAllocPrefix), nil), endCond)
}

// write all records into the database under given prefix
func write(writerChan chan rawEntry, errChan chan error, iter ethdb.Iterator, condition func(key []byte) (bool, error)) error {
	defer iter.Release()

	for iter.Next() {
		if condition != nil {
			finished, err := condition(iter.Key())
			if err != nil {
				return err
			}
			if finished {
				return nil
			}
		}

		select {
		case err, ok := <-errChan:
			{
				if ok {
					return err
				}
			}
		case writerChan <- rawEntry{Key: iter.Key(), Value: iter.Value()}:

		}
	}
	return nil
}

// writeDataAsync copies data from channel into targetDb
func writeDataAsync(targetDb ethdb.Database) (chan rawEntry, chan error) {
	writeChan := make(chan rawEntry, dbItemChanSize)
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		dbBatchWriter := targetDb.NewBatch()

		for {
			// do we have another available item?
			item, ok := <-writeChan
			if !ok {
				return
			}
			// iteration completed - finish write rest of the pending data
			if dbBatchWriter.ValueSize() > 0 {
				err := dbBatchWriter.Write()
				if err != nil {
					errChan <- err
					return
				}
			}

			key := item.Key

			err := dbBatchWriter.Put(key, item.Value)
			if err != nil {
				errChan <- err
				return
			}

			// writing data in batches
			if dbBatchWriter.ValueSize() > kvdb.IdealBatchSize {
				err = dbBatchWriter.Write()
				if err != nil {
					errChan <- err
					return
				}
				dbBatchWriter.Reset()
			}
		}
	}()
	return writeChan, errChan
}

// openCloneDatabases prepares aida and target databases
func openCloneDatabases(cfg *utils.Config) (ethdb.Database, ethdb.Database, error) {
	_, err := os.Stat(cfg.AidaDb)
	if os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("specified aida-db %v is empty\n", cfg.AidaDb)
	}

	_, err = os.Stat(cfg.TargetDb)
	if !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("specified target-db %v already exists\n", cfg.TargetDb)
	}

	// open aidaDb
	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return nil, nil, fmt.Errorf("targetDB. Error: %v", err)
	}

	// open targetDB
	targetDb, err := rawdb.NewLevelDBDatabase(cfg.TargetDb, 1024, 100, "profiling", false)
	if err != nil {
		return nil, nil, fmt.Errorf("targetDB. Error: %v", err)
	}

	return aidaDb, targetDb, nil
}
