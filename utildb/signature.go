package utildb

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/utils/dbcompoment"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
)

// DbSignature generates a signature for given dbcomponent
func DbSignature(cfg *utils.Config, aidaDb ethdb.Database, log logger.Logger) error {
	if *cfg.DbComponent == dbcompoment.Substate || *cfg.DbComponent == dbcompoment.All {
		log.Info("Generating Substate hash...")
		aidaDbSubstateHash, count, err := GetSubstateHash(cfg, aidaDb, log)
		if err != nil {
			return err
		}
		log.Infof("Substate hash: %x; count %v", aidaDbSubstateHash, count)
	}

	if *cfg.DbComponent == dbcompoment.Delete || *cfg.DbComponent == dbcompoment.All {
		log.Info("Generating Deletion hash...")
		aidaDbDeletionHash, count, err := GetDeletionHash(cfg, aidaDb, log)
		if err != nil {
			return err
		}
		log.Infof("Deletion hash: %x; count %v", aidaDbDeletionHash, count)
	}

	if *cfg.DbComponent == dbcompoment.Update || *cfg.DbComponent == dbcompoment.All {
		log.Info("Generating Updateset hash...")
		aidaDbUpdateDbHash, count, err := GetUpdateDbHash(cfg, aidaDb, log)
		if err != nil {
			return err
		}
		log.Infof("Updateset hash: %x; count %v", aidaDbUpdateDbHash, count)
	}

	if *cfg.DbComponent == dbcompoment.StateHash || *cfg.DbComponent == dbcompoment.All {
		log.Info("Generating State-Hashes hash...")
		aidaDbStateHashesHash, count, err := GetStateHashesHash(cfg, aidaDb, log)
		if err != nil {
			return err
		}
		log.Infof("State-Hashes hash: %x; count %v", aidaDbStateHashesHash, count)
	}

	return nil
}

// combineJson reads objects from in channel, encodes their []byte representation and writes to out channel
func combineJson(in chan any, out chan []byte, errChan chan error) {
	for {
		select {
		case value, ok := <-in:
			if !ok {
				return
			}
			jsonData, err := json.Marshal(value)
			if err != nil {
				errChan <- err
				return
			}
			out <- jsonData
		}
	}
}

func GetSubstateHash(cfg *utils.Config, db ethdb.Database, log logger.Logger) ([]byte, uint64, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	feeder := func(feederChan chan any, errChan chan error) {
		defer close(feederChan)

		substate.SetSubstateDbBackend(db)
		it := substate.NewSubstateIterator(0, 10)
		defer it.Release()

		for it.Next() {
			select {
			case <-ticker.C:
				log.Infof("SubstateDb hash progress: %v/%v", it.Value().Block, cfg.Last)
			default:
			}

			select {
			case err := <-errChan:
				errChan <- err
				return
			case feederChan <- it.Value():
			}
		}
	}

	return parallelHashComputing(feeder)
}

func GetDeletionHash(cfg *utils.Config, db ethdb.Database, log logger.Logger) ([]byte, uint64, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	feeder := func(feederChan chan any, errChan chan error) {
		defer close(feederChan)

		ddb := substate.NewDestroyedAccountDB(db)
		inRange, err := ddb.GetAccountsDestroyedInRange(0, cfg.Last)
		if err != nil {
			errChan <- err
			return
		}

		sort.Slice(inRange, func(i, j int) bool {
			return bytes.Compare(inRange[i].Bytes(), inRange[j].Bytes()) < 0
		})

		for i, address := range inRange {
			select {
			case <-ticker.C:
				log.Infof("DeletionDb hash progress: %v/%v", i, len(inRange))
			default:
			}

			select {
			case err = <-errChan:
				errChan <- err
				return
			case feederChan <- address.Hex():
			}
		}
	}
	return parallelHashComputing(feeder)
}

func GetUpdateDbHash(cfg *utils.Config, db ethdb.Database, log logger.Logger) ([]byte, uint64, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	feeder := func(feederChan chan any, errChan chan error) {
		defer close(feederChan)

		udb := substate.NewUpdateDB(db)
		iter := substate.NewUpdateSetIterator(udb, 0, cfg.Last)
		defer iter.Release()

		for iter.Next() {
			select {
			case <-ticker.C:
				log.Infof("UpdateDb hash progress: %v/%v", iter.Value().Block, cfg.Last)
			default:
			}

			value := iter.Value()
			select {
			case err := <-errChan:
				errChan <- err
				return
			case feederChan <- value.UpdateSet:
			}
		}
	}
	return parallelHashComputing(feeder)
}

func GetStateHashesHash(cfg *utils.Config, db ethdb.Database, log logger.Logger) ([]byte, uint64, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	feeder := func(feederChan chan any, errChan chan error) {
		defer close(feederChan)

		provider := utils.MakeStateHashProvider(db)

		var i uint64 = 0
		for ; i <= cfg.Last; i++ {
			select {
			case <-ticker.C:
				log.Infof("Stat-Hashes hash progress: %v/%v", i, cfg.Last)
			default:
			}

			h, err := provider.GetStateHash(int(i))
			if err != nil {
				if errors.Is(err, leveldb.ErrNotFound) {
					continue
				}
				errChan <- err
				return
			}

			select {
			case err = <-errChan:
				errChan <- err
				return
			case feederChan <- h:
			}
		}
	}

	return parallelHashComputing(feeder)
}

func parallelHashComputing(feeder func(chan any, chan error)) ([]byte, uint64, error) {
	var wg sync.WaitGroup
	feederChan := make(chan any, 1)
	processedChan := make(chan []byte, 1)

	errChan := make(chan error)

	var counter uint64 = 0

	countingFeeder := make(chan any)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case err := <-errChan:
				errChan <- err
				return
			case item, ok := <-countingFeeder:
				if !ok {
					close(feederChan)
					return
				}
				counter++

				select {
				case err := <-errChan:
					errChan <- err
					return
				case feederChan <- item:
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		feeder(countingFeeder, errChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		combineJson(feederChan, processedChan, errChan)
	}()

	// Start a goroutine to close hashChan when all workers finish
	go func() {
		wg.Wait()
		close(errChan)
		close(processedChan)
	}()

	hasher := md5.New()

	for {
		select {
		case err, ok := <-errChan:
			if ok {
				if err != nil {
					return nil, counter, err
				}
			}
		case value, ok := <-processedChan:
			if !ok {
				return hasher.Sum(nil), counter, nil
			}
			hasher.Write(value)
		}
	}
}
