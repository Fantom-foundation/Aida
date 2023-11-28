package utildb

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Aida/utils/dbcompoment"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTableHash_Empty(t *testing.T) {
	tmpDir := t.TempDir() + "/aidaDb"
	aidaDb, err := rawdb.NewLevelDBDatabase(tmpDir, 1024, 100, "aida-db", false)
	if err != nil {
		t.Fatalf("error opening leveldb %s: %v", tmpDir, err)
	}

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	// Create a config
	cfg := &utils.Config{
		DbComponent: new(dbcompoment.DbComponent), // Set this to the component you want to test
	}
	*cfg.DbComponent = dbcompoment.All

	gomock.InOrder(
		log.EXPECT().Info(gomock.Any()),
		log.EXPECT().Infof(gomock.Any(), gomock.Any(), uint64(0)),
		log.EXPECT().Info(gomock.Any()),
		log.EXPECT().Infof(gomock.Any(), gomock.Any(), uint64(0)),
		log.EXPECT().Info(gomock.Any()),
		log.EXPECT().Infof(gomock.Any(), gomock.Any(), uint64(0)),
		log.EXPECT().Info(gomock.Any()),
		log.EXPECT().Infof(gomock.Any(), gomock.Any(), uint64(0)),
	)

	// Call the function
	err = TableHash(cfg, aidaDb, log) // Pass a logger if needed
	assert.NoError(t, err)
}

func TestTableHash_Filled(t *testing.T) {
	tmpDir := t.TempDir() + "/aidaDb"
	aidaDb, err := rawdb.NewLevelDBDatabase(tmpDir, 1024, 100, "aida-db", false)
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	// Create a config
	cfg := &utils.Config{
		DbComponent: new(dbcompoment.DbComponent), // Set this to the component you want to test
		First:       0,
		Last:        100, // None of the following generators must not generate record higher than this number
	}
	*cfg.DbComponent = dbcompoment.All

	substateCount, deleteCount, updateCount, stateHashCount := fillFakeAidaDb(t, aidaDb)

	gomock.InOrder(
		log.EXPECT().Info(gomock.Any()),
		log.EXPECT().Infof(gomock.Any(), gomock.Any(), uint64(substateCount)),
		log.EXPECT().Info(gomock.Any()),
		log.EXPECT().Infof(gomock.Any(), gomock.Any(), uint64(deleteCount)),
		log.EXPECT().Info(gomock.Any()),
		log.EXPECT().Infof(gomock.Any(), gomock.Any(), uint64(updateCount)),
		log.EXPECT().Info(gomock.Any()),
		log.EXPECT().Infof(gomock.Any(), gomock.Any(), uint64(stateHashCount)),
	)

	// Call the function
	err = TableHash(cfg, aidaDb, log) // Pass a logger if needed
	assert.NoError(t, err)
}

func fillFakeAidaDb(t *testing.T, aidaDb ethdb.Database) (int, int, int, int) {
	// Seed the random number generator
	rand.NewSource(time.Now().UnixNano())

	substate.SetSubstateDbBackend(aidaDb)
	// Generate a random number between 1 and 5
	numSubstates := rand.Intn(5) + 1

	for i := 0; i < numSubstates; i++ {
		state := substate.Substate{
			Env: &substate.SubstateEnv{Number: uint64(i)},
			Message: &substate.SubstateMessage{
				Value: big.NewInt(int64(rand.Intn(100))),
			},
			InputAlloc:  substate.SubstateAlloc{},
			OutputAlloc: substate.SubstateAlloc{},
			Result:      &substate.SubstateResult{},
		}

		substate.PutSubstate(uint64(i), 0, &state)
	}

	ddb := substate.NewDestroyedAccountDB(aidaDb)

	// Generate random number between 6-10
	numDestroyedAccounts := rand.Intn(5) + 6

	for i := 0; i < numDestroyedAccounts; i++ {
		destroyedAccounts := []common.Address{
			common.BytesToAddress(utils.MakeRandomByteSlice(t, 40)),
		}
		err := ddb.SetDestroyedAccounts(uint64(i), 0, destroyedAccounts, []common.Address{})
		if err != nil {
			t.Fatalf("error setting destroyed accounts: %v", err)
		}
	}

	udb := substate.NewUpdateDB(aidaDb)

	// Generate random number between 11-15
	numUpdates := rand.Intn(5) + 11

	for i := 0; i < numUpdates; i++ {
		sa := new(substate.SubstateAccount)
		sa.Balance = big.NewInt(int64(utils.GetRandom(1, 1000*5000)))
		randomAddress := common.BytesToAddress(utils.MakeRandomByteSlice(t, 40))
		updateSet := substate.SubstateAlloc{
			randomAddress: sa,
		}
		udb.PutUpdateSet(uint64(i), &updateSet, []common.Address{})
	}

	// Generate random number between 16-20
	numStateHashes := rand.Intn(5) + 16

	for i := 0; i < numStateHashes; i++ {
		err := utils.SaveStateRoot(aidaDb, fmt.Sprintf("0x%x", i), strings.Repeat("0", 64))
		if err != nil {
			t.Fatalf("error saving state root: %v", err)
		}
	}

	return numSubstates, numDestroyedAccounts, numUpdates, numStateHashes
}
