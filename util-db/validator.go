package util_db

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	standardInputBufferSize = 50
	FirstOperaTestnetBlock  = 479327
)

// validator is used to iterate over all key/value pairs inside AidaDb and creating md5 hash
type validator struct {
	db     ethdb.Database
	start  time.Time
	input  chan []byte
	result chan []byte
	closed chan any
	log    logger.Logger
	wg     *sync.WaitGroup
}

// FindDbHashOnline if user has no dbHash inside his AidaDb metadata
func FindDbHashOnline(chainId utils.ChainID, log logger.Logger, md *utils.AidaDbMetadata) ([]byte, error) {
	var url string

	if chainId == utils.MainnetChainID {
		url = utils.AidaDbRepositoryMainnetUrl
	} else if chainId == utils.TestnetChainID {
		url = utils.AidaDbRepositoryTestnetUrl
	}

	log.Noticef("looking for db-hash online on %v", url)
	patches, err := utils.DownloadPatchesJson()
	if err != nil {
		return nil, err
	}

	md.LastBlock = md.GetLastBlock()

	if md.LastBlock == 0 {
		log.Warning("your aida-db seems to have empty metadata; looking for block range in substate")
	}

	var ok bool

	md.FirstBlock, md.LastBlock, ok = utils.FindBlockRangeInSubstate()
	if !ok {
		return nil, errors.New("cannot find block range in substate")
	}

	err = md.SetBlockRange(md.FirstBlock, md.LastBlock)
	if err != nil {
		return nil, err
	}

	for _, patch := range patches {
		if patch.ToBlock == md.LastBlock {
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

// GenerateDbHash for given AidaDb
func GenerateDbHash(db ethdb.Database, logLevel string) ([]byte, error) {
	v := newDbValidator(db, logLevel)

	v.wg.Add(2)

	go v.calculate()
	go v.iterate()

	var sum []byte

	select {
	case sum = <-v.result:
		v.log.Notice("DbHash Generation complete!")
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

	v.log.Notice("Iterating over State Hashes...")
	v.doIterate(utils.StateHashPrefix)

	v.log.Infof("State Hashes took %v.", time.Since(now).Round(1*time.Second))

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
