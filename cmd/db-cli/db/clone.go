package db

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const cloneWriteChanSize = 1

// CloneCommand enables creation of aida-db read or subset
var CloneCommand = cli.Command{
	Action: clone,
	Name:   "clone",
	Usage:  "clone can create aida-db read or subset",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.TargetDbFlag,
		&utils.CompactDbFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates clone of aida-db for desired block range
`,
}

type cloner struct {
	cfg             *utils.Config
	log             *logging.Logger
	aidaDb, cloneDb ethdb.Database
	count           uint64
	writeCh         chan rawEntry
	errCh           chan error
	closeCh         chan any
}

// rawEntry representation of database entry
type rawEntry struct {
	Key   []byte
	Value []byte
}

// clone creates aida-db copy or subset
func clone(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	log := logger.NewLogger(cfg.LogLevel, "AidaDb Clone")

	c := cloner{
		cfg:     cfg,
		log:     log,
		writeCh: make(chan rawEntry, cloneWriteChanSize),
		errCh:   make(chan error, 1),
		closeCh: make(chan any),
	}

	if err = c.openDbs(); err != nil {
		return err
	}

	if err = c.clone(); err != nil {
		return err
	}

	MustCloseDB(c.aidaDb)
	MustCloseDB(c.cloneDb)

	err = ctx.Set(utils.AidaDbFlag.Name, cfg.TargetDb)
	if err != nil {
		return err
	}
	return printMetadata(c.cfg.TargetDb)
}

// openDbs prepares aida and target databases
func (c *cloner) openDbs() error {
	var err error

	_, err = os.Stat(c.cfg.AidaDb)
	if os.IsNotExist(err) {
		return fmt.Errorf("specified aida-db %v is empty\n", c.cfg.AidaDb)
	}

	_, err = os.Stat(c.cfg.TargetDb)
	if !os.IsNotExist(err) {
		return fmt.Errorf("specified target-db %v already exists\n", c.cfg.TargetDb)
	}

	// open db
	c.aidaDb, err = rawdb.NewLevelDBDatabase(c.cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("targetDb; %v", err)
	}

	// open cloneDb
	c.cloneDb, err = rawdb.NewLevelDBDatabase(c.cfg.TargetDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("targetDb; %v", err)
	}

	return nil
}

// clone AidaDb in given block range
func (c *cloner) clone() error {
	go c.write()
	go c.checkErrors()

	c.read([]byte(substate.Stage1CodePrefix), 0, nil)
	c.read([]byte(substate.DestroyedAccountPrefix), 0, nil)
	lastUpdateBeforeRange := c.readUpdateSet()
	c.readSubstate(lastUpdateBeforeRange)

	close(c.writeCh)

	processCloneLikeMetadata(c.cloneDb, c.cfg.LogLevel, c.cfg.First, c.cfg.Last, c.cfg.ChainID)

	//  compact written data
	if c.cfg.CompactDb {
		c.log.Noticef("Starting compaction")
		err := c.cloneDb.Compact(nil, nil)
		if err != nil {
			return err
		}
	}

	if c.cfg.Validate {
		err := c.validateDbSize()
		if err != nil {
			return err
		}
	}

	return nil
}

// checkErrors is a thread for error handling. When error occurs in any thread, this thread closes every other thread
func (c *cloner) checkErrors() {
	for {
		select {
		case <-c.closeCh:
			return
		case err := <-c.errCh:
			c.log.Fatal(err)
			c.stop()
			return
		}
	}
}

// write data read from func read() into new cloneDb
func (c *cloner) write() {
	var (
		err         error
		data        rawEntry
		ok          bool
		batchWriter ethdb.Batch
	)

	batchWriter = c.cloneDb.NewBatch()

	for {
		select {
		case data, ok = <-c.writeCh:
			if !ok {
				// iteration completed - read rest of the pending data
				if batchWriter.ValueSize() > 0 {
					err = batchWriter.Write()
					if err != nil {
						c.errCh <- fmt.Errorf("cannot read rest of the data into cloneDb; %v", err)
						return
					}
				}
				return
			}

			err = batchWriter.Put(data.Key, data.Value)
			if err != nil {
				c.errCh <- fmt.Errorf("cannot put data into cloneDb %v", err)
				return
			}

			// writing data in batches
			if batchWriter.ValueSize() > kvdb.IdealBatchSize {
				err = batchWriter.Write()
				if err != nil {
					c.errCh <- fmt.Errorf("cannot write batch; %v", err)
					return
				}

				// reset writer after writing batch
				batchWriter.Reset()
			}
		case <-c.closeCh:
			return
		}

	}
}

// read data with given prefix until given condition is fulfilled from source AidaDb
func (c *cloner) read(prefix []byte, start uint64, condition func(key []byte) (bool, error)) {
	c.log.Noticef("Copying data with prefix %v", string(prefix))

	iter := c.aidaDb.NewIterator(prefix, substate.BlockToBytes(start))
	defer iter.Release()

	for iter.Next() {
		if condition != nil {
			finished, err := condition(iter.Key())
			if err != nil {
				c.errCh <- fmt.Errorf("condition emit error; %v", err)
				return
			}
			if finished {
				break
			}
		}

		c.count++

		// make deep read key and value
		// need to pass deep read of values into the channel
		// golang channels were using pointers and values read from channel were incorrect
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())

		select {
		case <-c.closeCh:
			return
		case c.writeCh <- rawEntry{Key: key, Value: value}:
		}
	}

	c.log.Noticef("Prefix %v done", string(prefix))

	return
}

// readUpdateSet from UpdateDb
func (c *cloner) readUpdateSet() uint64 {
	// labeling last updateSet before interval - need to export substate for that range as well
	var lastUpdateBeforeRange uint64
	endCond := func(key []byte) (bool, error) {
		block, err := substate.DecodeUpdateSetKey(key)
		if err != nil {
			return false, err
		}
		if block > c.cfg.Last {
			return true, nil
		}
		if block < c.cfg.First {
			lastUpdateBeforeRange = block
		}
		return false, nil
	}

	c.read([]byte(substate.SubstateAllocPrefix), 0, endCond)

	// check if update-set contained at least one set (first set with world-state), then aida-db must be corrupted
	if lastUpdateBeforeRange == 0 {

		c.errCh <- fmt.Errorf("updateset didn't contain any records - unable to create aida-db without initial world-state")
		return 0
	}

	return lastUpdateBeforeRange
}

// readSubstate from last updateSet before cfg.First until cfg.Last
func (c *cloner) readSubstate(lastUpdateBeforeRange uint64) {
	endCond := func(key []byte) (bool, error) {
		block, _, err := substate.DecodeStage1SubstateKey(key)
		if err != nil {
			return false, err
		}
		if block > c.cfg.Last {
			return true, nil
		}
		return false, nil
	}

	// generating substate right after previous updateSet before our interval
	c.read([]byte(substate.Stage1SubstatePrefix), lastUpdateBeforeRange+1, endCond)
	return
}

// validateDbSize compares size of database and expectedWritten
func (c *cloner) validateDbSize() error {
	actualWritten := getDbSize(c.cloneDb)
	if actualWritten != c.count {
		return fmt.Errorf("TargetDb has %v records; expected: %v", actualWritten, c.count)
	}
	return nil
}

// closeDbs when cloning is done
func (c *cloner) closeDbs() {
	var err error

	if err = c.aidaDb.Close(); err != nil {
		c.log.Errorf("cannot close aida-db")
	}

	if err = c.cloneDb.Close(); err != nil {
		c.log.Errorf("cannot close aida-db")
	}
}

// stop all cloner threads
func (c *cloner) stop() {
	select {
	case <-c.closeCh:
		return
	default:
		close(c.closeCh)
		c.closeDbs()
	}
}
