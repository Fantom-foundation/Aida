package db

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

const standardInputBufferSize = 50

var ValidateCommand = cli.Command{
	Action: validateCmd,
	Name:   "validate",
	Usage:  "Validates AidaDb using md5 DbHash.",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
	},
}

// validator is used to iterate over all key/value pairs inside AidaDb and creating md5 hash
type validator struct {
	db     ethdb.Database
	start  time.Time
	input  chan []byte
	result chan []byte
	closed chan any
	log    *logging.Logger
	wg     *sync.WaitGroup
}

// validateCmd calculates the dbHash for given AidaDb and compares it to expected hash either found in metadata or online
func validateCmd(ctx *cli.Context) error {
	log := logger.NewLogger("INFO", "ValidateCMD")

	cfg, err := utils.NewConfig(ctx, utils.NoArgs)

	aidaDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	defer MustCloseDB(aidaDb)

	md := utils.NewAidaDbMetadata(aidaDb, "INFO")

	// validation only makes sense if user has pure AidaDb
	dbType := md.GetDbType()
	if dbType != utils.GenType {
		return fmt.Errorf("validation cannot be performed - your db type (%v) cannot be validated; aborting", dbType)
	}

	// we need to make sure aida-db starts from beginning, otherwise validation is impossible
	fb := md.GetFirstBlock()
	if fb != 0 {
		return fmt.Errorf("validation cannot be performed - your db does not start at block 0; your first block: %v", fb)
	}

	var saveHash = false

	// if db hash is not present, look for it in patches.json
	expectedHash := md.GetDbHash()
	if len(expectedHash) == 0 {
		// we want to save the hash inside metadata
		saveHash = true
		expectedHash, err = findDbHashOnline(cfg, log, md)
		if err != nil {
			return fmt.Errorf("validation cannot be performed; %v", err)
		}
	}

	log.Noticef("Starting DbHash calculation for %v; this may take several hours...", cfg.AidaDb)
	trueHash, err := validate(aidaDb, "INFO")
	if err != nil {
		return err
	}

	if bytes.Compare(expectedHash, trueHash) != 0 {
		return fmt.Errorf("hashes are different! expected: %v; your aida-db:%v", hex.EncodeToString(expectedHash), hex.EncodeToString(trueHash))
	}

	log.Noticef("Validation successful!")

	if saveHash {
		err = md.SetDbHash(trueHash)
		if err != nil {
			return err
		}
	}

	return nil
}

// findDbHashOnline if user has no dbHash inside his AidaDb metadata
func findDbHashOnline(cfg *utils.Config, log *logging.Logger, md *utils.AidaDbMetadata) ([]byte, error) {
	var (
		url     string
		chainId int
	)

	chainId = md.GetChainID()
	if chainId == 0 {
		log.Warning("cannot find db-hash in your aida-db metadata, this operation is needed because db-hash was not found inside your aida-db; please make sure you specified correct chain-id with flag --%v", utils.ChainIDFlag.Name)
		chainId = cfg.ChainID
	}

	if chainId == 250 {
		url = utils.AidaDbRepositoryMainnetUrl
	} else if chainId == 4002 {
		url = utils.AidaDbRepositoryTestnetUrl
	}

	log.Noticef("looking for db-hash online on %v", url)
	patches, err := downloadPatchesJson()
	if err != nil {
		return nil, err
	}

	lb := md.GetLastBlock()

	if lb == 0 {
		log.Warning("your aida-db seems to have empty metadata; looking for block range in substate")
	}

	fb, lb, ok := utils.FindBlockRangeInSubstate()
	if !ok {
		return nil, errors.New("cannot find block range in substate")
	}

	err = md.SetBlockRange(fb, lb)
	if err != nil {
		return nil, err
	}

	for _, patch := range patches {
		if patch.ToBlock == lb {
			return hex.DecodeString(patch.DbHash)
		}
	}

	return nil, errors.New("could not find db-hash for your db range")
}

// newDbValidator returns new instance of validator
func newDbValidator(db ethdb.Database, logLevel string) *validator {
	l := logger.NewLogger(logLevel, "Db-Validator")

	return &validator{
		closed: make(chan any, 1),
		db:     db,
		input:  make(chan []byte, standardInputBufferSize),
		result: make(chan []byte, 1),
		start:  time.Now(),
		log:    l,
		wg:     new(sync.WaitGroup),
	}
}

// validate AidaDb on given path pathToDb
func validate(db ethdb.Database, logLevel string) ([]byte, error) {
	v := newDbValidator(db, logLevel)

	v.wg.Add(2)

	go v.calculate()
	go v.iterate()

	var sum []byte

	select {
	case sum = <-v.result:
		v.log.Noticef("AidaDb MD5 sum: %v", hex.EncodeToString(sum))
		break
	case <-v.closed:
		break
	}

	v.wg.Wait()
	return sum, nil
}

// iterate calls doIterate func for each prefix inside metadata
func (v *validator) iterate() {
	var now time.Time

	defer func() {
		close(v.input)
		v.wg.Done()
	}()

	now = time.Now()

	v.log.Notice("Iterating over Stage 1 Substate...")
	v.doIterate(substate.Stage1SubstatePrefix)

	v.log.Infof("Stage 1 Substate took %v.", time.Since(now).Round(1*time.Second))

	now = time.Now()

	v.log.Notice("Iterating over Substate Alloc...")
	v.doIterate(substate.SubstateAllocPrefix)

	v.log.Infof("Substate Alloc took %v.", time.Since(now).Round(1*time.Second))

	now = time.Now()

	v.log.Notice("Iterating over Destroyed Accounts...")
	v.doIterate(substate.DestroyedAccountPrefix)

	v.log.Infof("Destroyed Accounts took %v.", time.Since(now).Round(1*time.Second))

	v.log.Noticef("Total time elapsed: %v", time.Since(v.start).Round(1*time.Second))

	return

}

// doIterate over all key/value inside AidaDb and create md5 hash for each par for given prefix
func (v *validator) doIterate(prefix string) {
	iter := v.db.NewIterator([]byte(prefix), nil)

	defer func() {
		iter.Release()
	}()

	var (
		dst, b []byte
	)

	for iter.Next() {
		b = iter.Key()
		dst = make([]byte, len(b))
		copy(dst, b)

		select {
		case <-v.closed:
			return
		case v.input <- dst:
			break
		}

		b = iter.Value()
		dst = make([]byte, len(b))
		copy(dst, b)

		select {
		case <-v.closed:
			return
		case v.input <- dst:
			break
		}
	}

	if iter.Error() != nil {
		v.stop()
		v.log.Errorf("cannot iterate; %v", iter.Error())
	}

	return
}

// stop sends stopping signal by closing the closed chanel
func (v *validator) stop() {
	select {
	case <-v.closed:
		return
	default:
		close(v.closed)
	}
}

// calculate receives data from input chanel and calculates hash for each key and value
func (v *validator) calculate() {
	var (
		in         []byte
		h          = md5.New()
		written, n int
		err        error
		ok         bool
	)

	defer func() {
		v.wg.Done()
	}()

	for {
		select {
		case <-v.closed:
			return
		case in, ok = <-v.input:
			if !ok {
				v.result <- h.Sum(nil)
				return
			}

			// we need to make sure we have written all the data
			for written < len(in) {
				n, err = h.Write(in[written:])
				written += n
			}

			// reset counter
			written = 0

			if err != nil {
				v.log.Criticalf("cannot write hash; %v", err)
				v.stop()
				return
			}

		}
	}
}
