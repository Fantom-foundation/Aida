package utildb

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
)

func DbSignature(cfg *utils.Config, aidaDb ethdb.Database, log logger.Logger) error {
	log.Info("Generating substate...")
	aidaDbSubstateHash, err := GetSubstateHash(aidaDb, cfg.Workers, log)
	if err != nil {
		return err
	}
	log.Infof("Substate hash: %x", aidaDbSubstateHash)

	log.Info("Generating deletion hash...")
	aidaDbDeletionHash, err := GetDeletionHash(cfg, aidaDb, log)
	if err != nil {
		return err
	}
	log.Infof("Deletion hash: %x", aidaDbDeletionHash)

	log.Info("Generating updateDb hash...")
	aidaDbUpdateDbHash, err := GetUpdateDbHash(cfg, aidaDb, log)
	if err != nil {
		return err
	}
	log.Infof("UpdateDb hash: %x", aidaDbUpdateDbHash)

	log.Info("Generating state hashes hash...")
	aidaDbStateHashesHash, err := GetStateHashesHash(cfg, aidaDb, log)
	if err != nil {
		return err
	}
	log.Infof("State hashes hash: %x", aidaDbStateHashesHash)

	return nil
}

func marshaller(in chan any, out chan []byte, errChan chan error) {
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

func GetSubstateHash(db ethdb.Database, workers int, log logger.Logger) ([]byte, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	feeder := func(feederChan chan any, errChan chan error) {
		substate.SetSubstateDbBackend(db)
		it := substate.NewSubstateIterator(0, workers)
		defer it.Release()

		for it.Next() {
			select {
			case <-ticker.C:
				log.Infof("Substate hash: %v", it.Value().Block)
			default:
			}

			select {
			case err, ok := <-errChan:
				if ok {
					if err != nil {
						errChan <- err
						return
					}
				}
			case feederChan <- it.Value():
			}
		}
	}

	return parallelHashComputing(feeder, workers)
}

func GetDeletionHash(cfg *utils.Config, db ethdb.Database, log logger.Logger) ([]byte, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	feeder := func(feederChan chan any, errChan chan error) {
		ddb := substate.NewDestroyedAccountDB(db)
		inRange, err := ddb.GetAccountsDestroyedInRange(0, cfg.Last)
		if err != nil {
			errChan <- err
			return
		}

		for i, address := range inRange {
			select {
			case <-ticker.C:
				log.Infof("Deletion hash at: %v/%v", i, len(inRange))
			default:
			}

			select {
			case err, ok := <-errChan:
				if ok {
					if err != nil {
						errChan <- err
						return
					}
				}
			case feederChan <- address.Bytes():
			}
		}
	}
	return parallelHashComputing(feeder, cfg.Workers)
}

func GetUpdateDbHash(cfg *utils.Config, db ethdb.Database, log logger.Logger) ([]byte, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	feeder := func(feederChan chan any, errChan chan error) {
		udb := substate.NewUpdateDB(db)
		iter := substate.NewUpdateSetIterator(udb, 0, cfg.Last)
		defer iter.Release()

		for iter.Next() {
			select {
			case <-ticker.C:
				log.Infof("UpdateDb hash at: %v/%v", iter.Value().Block, cfg.Last)
			default:
			}

			value := iter.Value()
			select {
			case err, ok := <-errChan:
				if ok {
					if err != nil {
						errChan <- err
						return
					}
				}
			case feederChan <- value:
			}
		}
	}
	//using 1 worker to avoid memory issues
	return parallelHashComputing(feeder, 1)
}

func GetStateHashesHash(cfg *utils.Config, db ethdb.Database, log logger.Logger) ([]byte, error) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	feeder := func(feederChan chan any, errChan chan error) {
		provider := utils.MakeStateHashProvider(db)

		var i uint64 = 0
		for ; i < cfg.Last; i++ {
			select {
			case <-ticker.C:
				log.Infof("State hashes hash at: %v/%v", i, cfg.Last)
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
			case err, ok := <-errChan:
				if ok {
					if err != nil {
						errChan <- err
						return
					}
				}
			case feederChan <- h:
			}
		}
	}

	return parallelHashComputing(feeder, cfg.Workers)
}

func parallelHashComputing(feeder func(chan any, chan error), workers int) ([]byte, error) {
	var wg sync.WaitGroup
	feederChan := make(chan any, workers)
	processedChan := make(chan []byte, workers)

	errChan := make(chan error)

	go func() {
		defer close(feederChan)

		wg.Add(1)
		defer wg.Done()

		feeder(feederChan, errChan)
	}()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			marshaller(feederChan, processedChan, errChan)
		}()

	}

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
					return nil, err
				}
			}
		case value, ok := <-processedChan:
			if !ok {
				return hasher.Sum(nil), nil
			}
			hasher.Write(value)
		}
	}
}
