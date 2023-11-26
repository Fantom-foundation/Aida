package utildb

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/ethdb"
)

// FindBlockRangeInUpdate finds the first and last block in the update set
func FindBlockRangeInUpdate(aidaDb ethdb.Database) (uint64, uint64, error) {
	udb := substate.NewUpdateDB(aidaDb)
	firstBlock, err := udb.GetFirstKey()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get first updateset; %v", err)
	}

	// get last updateset
	lastBlock, err := udb.GetLastKey()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get last updateset; %v", err)
	}
	return firstBlock, lastBlock, nil
}

// FindBlockRangeInDeleted finds the first and last block in the deleted accounts
func FindBlockRangeInDeleted(aidaDb ethdb.Database) (uint64, uint64, error) {
	ddb := substate.NewDestroyedAccountDB(aidaDb)
	firstBlock, err := ddb.GetFirstKey()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get first deleted accounts; %v", err)
	}

	// get last updateset
	lastBlock, err := ddb.GetLastKey()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get last deleted accounts; %v", err)
	}
	return firstBlock, lastBlock, nil
}

// FindBlockRangeInStateHash finds the first and last block in the state hash
func FindBlockRangeInStateHash(db ethdb.Database, log logger.Logger) (uint64, uint64, error) {
	var firstStateHashBlock, lastStateHashBlock uint64
	var err error
	firstStateHashBlock, err = utils.GetFirstStateHash(db)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get first state hash; %v", err)
	}

	lastStateHashBlock, err = utils.GetLastStateHash(db)
	if err != nil {
		log.Infof("Found first state hash at %v", firstStateHashBlock)
		return 0, 0, fmt.Errorf("cannot get last state hash; %v", err)
	}
	return firstStateHashBlock, lastStateHashBlock, nil
}
