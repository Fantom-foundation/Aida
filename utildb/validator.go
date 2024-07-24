// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utildb

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
)

const (
	standardInputBufferSize = 50
	FirstOperaTestnetBlock  = 479327
)

// validator is used to iterate over all key/value pairs inside AidaDb and creating md5 hash
type validator struct {
	db     db.BaseDB
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

	md.FirstBlock, md.LastBlock, ok = utils.FindBlockRangeInSubstate(db.MakeDefaultSubstateDBFromBaseDB(md.Db))
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
func newDbValidator(db db.BaseDB, logLevel string) *validator {
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
func GenerateDbHash(db db.BaseDB, logLevel string) ([]byte, error) {
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

// GeneratePrefixHash for given AidaDb
func GeneratePrefixHash(db db.BaseDB, prefix string, logLevel string) ([]byte, error) {
	v := newDbValidator(db, logLevel)

	v.wg.Add(2)

	go v.calculate()
	go func() {
		now := time.Now()
		v.log.Noticef("Iterating over Prefix %v...", prefix)
		count := v.doIterate(prefix)
		v.log.Infof("Prefix %v; has %v records; took %v.", prefix, count, time.Since(now).Round(1*time.Second))
		close(v.input)
		v.wg.Done()
	}()

	var sum []byte

	select {
	case sum = <-v.result:
		v.log.Noticef("Prefix %v ; MD5 sum: %v", prefix, hex.EncodeToString(sum))
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
	v.doIterate(db.SubstateDBPrefix)

	v.log.Infof("Stage 1 Substate took %v.", time.Since(now).Round(1*time.Second))

	now = time.Now()

	v.log.Notice("Iterating over Substate Alloc...")
	v.doIterate(db.UpdateDBPrefix)

	v.log.Infof("Substate Alloc took %v.", time.Since(now).Round(1*time.Second))

	now = time.Now()

	v.log.Notice("Iterating over Destroyed Accounts...")
	v.doIterate(db.DestroyedAccountPrefix)

	v.log.Infof("Destroyed Accounts took %v.", time.Since(now).Round(1*time.Second))

	v.log.Notice("Iterating over State Hashes...")
	v.doIterate(utils.StateHashPrefix)

	v.log.Infof("State Hashes took %v.", time.Since(now).Round(1*time.Second))

	v.log.Noticef("Total time elapsed: %v", time.Since(v.start).Round(1*time.Second))

	return

}

// doIterate over all key/value inside AidaDb and create md5 hash for each par for given prefix
func (v *validator) doIterate(prefix string) (count uint64) {
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
		count++
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
