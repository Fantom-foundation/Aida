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

// CloneCommand clones aida-db as standalone or patch database
var CloneCommand = cli.Command{
	Name:  "clone",
	Usage: `Used for creation of standalone subset of aida-db or patch`,
	Subcommands: []*cli.Command{
		&CloneDb,
		&ClonePatch,
	},
}

// ClonePatch enables creation of aida-db read or subset
var ClonePatch = cli.Command{
	Action:    clonePatch,
	Name:      "patch",
	Usage:     "patch is used to create aida-db patch",
	ArgsUsage: "<blockNumFirst> <blockNumLast> <EpochNumFirst> <EpochNumLast>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.TargetDbFlag,
		&utils.CompactDbFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates patch of aida-db for desired block range
`,
}

// CloneDb enables creation of aida-db read or subset
var CloneDb = cli.Command{
	Action:    createDbClone,
	Name:      "db",
	Usage:     "clone db creates aida-db subset",
	ArgsUsage: "<blockNumFirst> <blockNumLast>",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.TargetDbFlag,
		&utils.CompactDbFlag,
		&utils.ValidateFlag,
		&logger.LogLevelFlag,
	},
	Description: `
Creates clone db is used to create subset of aida-db to have more compact database, but still fully usable for desired block range.
`,
}

type cloner struct {
	cfg             *utils.Config
	log             *logging.Logger
	aidaDb, cloneDb ethdb.Database
	count           uint64
	typ             utils.AidaDbType
	writeCh         chan rawEntry
	errCh           chan error
	closeCh         chan any
}

// rawEntry representation of database entry
type rawEntry struct {
	Key   []byte
	Value []byte
}

// clonePatch creates aida-db patch
func clonePatch(ctx *cli.Context) error {
	// TODO refactor
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}

	if ctx.Args().Len() != 4 {
		return fmt.Errorf("clone patch command requires exactly 4 arguments")
	}

	cfg.First, cfg.Last, err = utils.SetBlockRange(ctx.Args().Get(0), ctx.Args().Get(1), cfg.ChainID)
	if err != nil {
		return err
	}

	var firstEpoch, lastEpoch uint64
	firstEpoch, lastEpoch, err = utils.SetBlockRange(ctx.Args().Get(2), ctx.Args().Get(3), cfg.ChainID)
	if err != nil {
		return err
	}

	aidaDb, targetDb, err := openCloningDbs(cfg.AidaDb, cfg.TargetDb)
	if err != nil {
		return err
	}

	err = CreatePatchClone(cfg, aidaDb, targetDb, firstEpoch, lastEpoch)
	if err != nil {
		return err
	}

	MustCloseDB(aidaDb)
	MustCloseDB(targetDb)

	return printMetadata(cfg.TargetDb)

}

// CreatePatchClone creates aida-db patch
func CreatePatchClone(cfg *utils.Config, aidaDb, targetDb ethdb.Database, firstEpoch uint64, lastEpoch uint64) error {
	err := clone(cfg, aidaDb, targetDb, utils.PatchType)
	if err != nil {
		return err
	}

	md := utils.NewAidaDbMetadata(targetDb, cfg.LogLevel)
	err = md.SetFirstEpoch(firstEpoch)
	if err != nil {
		return err
	}
	err = md.SetLastEpoch(lastEpoch)
	if err != nil {
		return err
	}

	return nil
}

// createDbClone creates aida-db copy or subset
func createDbClone(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.BlockRangeArgs)
	if err != nil {
		return err
	}

	aidaDb, targetDb, err := openCloningDbs(cfg.AidaDb, cfg.TargetDb)
	if err != nil {
		return err
	}

	err = clone(cfg, aidaDb, targetDb, utils.CloneType)
	if err != nil {
		return err
	}

	MustCloseDB(aidaDb)
	MustCloseDB(targetDb)

	return printMetadata(cfg.TargetDb)
}

func clone(cfg *utils.Config, aidaDb, cloneDb ethdb.Database, cloneType utils.AidaDbType) error {
	var err error
	log := logger.NewLogger(cfg.LogLevel, "AidaDb Clone")

	c := cloner{
		cfg:     cfg,
		cloneDb: cloneDb,
		aidaDb:  aidaDb,
		log:     log,
		typ:     cloneType,
		writeCh: make(chan rawEntry, cloneWriteChanSize),
		errCh:   make(chan error, 1),
		closeCh: make(chan any),
	}

	if err = c.clone(); err != nil {
		return err
	}

	return nil
}

// createDbClone AidaDb in given block range
func (c *cloner) clone() error {
	go c.write()
	go c.checkErrors()

	c.read([]byte(substate.Stage1CodePrefix), 0, nil)

	// update c.cfg.First block before loading deletions and substates, because for utils.CloneType those are necessery to be from last updateset onward
	lastUpdateBeforeRange := c.readUpdateSet()
	if c.typ == utils.CloneType {
		if lastUpdateBeforeRange < c.cfg.First {
			c.log.Noticef("Last updateset found at block %v, changing first block to %v", lastUpdateBeforeRange, lastUpdateBeforeRange+1)
			c.cfg.First = lastUpdateBeforeRange + 1
		}
	}

	err := c.readDeletions()
	if err != nil {
		return err
	}
	err = c.readSubstate()
	if err != nil {
		return err
	}

	close(c.writeCh)

	sourceMD := utils.NewAidaDbMetadata(c.aidaDb, c.cfg.LogLevel)
	chainID := sourceMD.GetChainID()

	if err = utils.ProcessCloneLikeMetadata(c.cloneDb, c.typ, c.cfg.LogLevel, c.cfg.First, c.cfg.Last, chainID); err != nil {
		return err
	}

	//  compact written data
	if c.cfg.CompactDb {
		c.log.Noticef("Starting compaction")
		err = c.cloneDb.Compact(nil, nil)
		if err != nil {
			return err
		}
	}

	if c.cfg.Validate {
		err = c.validateDbSize()
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

// write data read from func read() into new createDbClone
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
						c.errCh <- fmt.Errorf("cannot read rest of the data into createDbClone; %v", err)
						return
					}
				}
				return
			}

			err = batchWriter.Put(data.Key, data.Value)
			if err != nil {
				c.errCh <- fmt.Errorf("cannot put data into createDbClone %v", err)
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

	if c.typ == utils.CloneType {
		c.read([]byte(substate.SubstateAllocPrefix), 0, endCond)

		// check if update-set contained at least one set (first set with world-state), then aida-db must be corrupted
		if lastUpdateBeforeRange == 0 {
			c.errCh <- fmt.Errorf("updateset didn't contain any records - unable to create aida-db without initial world-state")
			return 0
		}

		return lastUpdateBeforeRange
	} else if c.typ == utils.PatchType {
		c.read([]byte(substate.SubstateAllocPrefix), c.cfg.First, endCond)
		return 0
	} else {
		c.errCh <- fmt.Errorf("incorrect clone type: %v", c.typ)
		return 0
	}
}

// readSubstate from last updateSet before cfg.First until cfg.Last
func (c *cloner) readSubstate() error {
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

	c.read([]byte(substate.Stage1SubstatePrefix), c.cfg.First, endCond)

	return nil
}

// readDeletions from last updateSet before cfg.First until cfg.Last
func (c *cloner) readDeletions() error {
	endCond := func(key []byte) (bool, error) {
		block, _, err := substate.DecodeDestroyedAccountKey(key)
		if err != nil {
			return false, err
		}
		if block > c.cfg.Last {
			return true, nil
		}
		return false, nil
	}

	c.read([]byte(substate.DestroyedAccountPrefix), c.cfg.First, endCond)

	return nil
}

// validateDbSize compares size of database and expectedWritten
func (c *cloner) validateDbSize() error {
	actualWritten := GetDbSize(c.cloneDb)
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

// openCloningDbs prepares aida and target databases
func openCloningDbs(aidaDbPath, targetDbPath string) (ethdb.Database, ethdb.Database, error) {
	var err error

	// if source db doesn't exist raise error
	_, err = os.Stat(aidaDbPath)
	if os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("specified aida-db %v is empty\n", aidaDbPath)
	}

	// if target db exists raise error
	_, err = os.Stat(targetDbPath)
	if !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("specified target-db %v already exists\n", targetDbPath)
	}

	var aidaDb, cloneDb ethdb.Database

	// open db
	aidaDb, err = rawdb.NewLevelDBDatabase(aidaDbPath, 1024, 100, "profiling", true)
	if err != nil {
		return nil, nil, fmt.Errorf("aidaDb %v; %v", aidaDbPath, err)
	}

	// open createDbClone
	cloneDb, err = rawdb.NewLevelDBDatabase(targetDbPath, 1024, 100, "profiling", false)
	if err != nil {
		return nil, nil, fmt.Errorf("targetDb %v; %v", targetDbPath, err)
	}

	return aidaDb, cloneDb, nil
}
