package db

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/op/go-logging"
)

const (
	pathToAidaTestDb  = "/var/opera/Aida/test/aida-db"
	pathToGenesis     = "/var/opera/Aida/test/genesis/testnet-2458-pruned-mpt.g"
	pathToOperaTestDb = "/var/opera/Aida/test/opera-db"
	expectedDbHash    = "7d858015ddff215cc2dce2a862a7b750"
)

// TestAutogen by creating one epoch large AidaDb and OperaDb.
// Testnet genesis file is needed for this test.
func TestAutogen(t *testing.T) {

	ctx, cfg := utils.CreateTestEnvironment()

	cfg.AidaDb = pathToAidaTestDb
	cfg.Db = pathToOperaTestDb
	cfg.Genesis = pathToGenesis

	g, err := newGenerator(ctx, cfg)
	if err != nil {
		t.Error(err)
		return
	}

	err = g.opera.init()
	if err != nil {
		t.Error(err)
		return
	}

	// remove worldstate directory if it was created
	defer func(log *logging.Logger) {
		if cfg.WorldStateDb != "" {
			err = os.RemoveAll(cfg.WorldStateDb)
			if err != nil {
				log.Criticalf("can't remove temporary folder: %v; %v", cfg.WorldStateDb, err)
			}
		}

		// remove both aida and opera db after testing
		err = os.RemoveAll(g.cfg.AidaDb)
		if err != nil {
			t.Error(err)
			return
		}

		err = os.RemoveAll(g.cfg.Db)
		if err != nil {
			t.Error(err)
			return
		}
	}(g.log)

	err = g.calculatePatchEnd()
	if err != nil {
		t.Error(err)
		return
	}

	// set range only on one epoch
	g.stopAtEpoch = g.opera.lastEpoch + 1

	g.log.Noticef("Starting substate generation %d - %d", g.opera.lastEpoch+1, g.stopAtEpoch)

	MustCloseDB(g.aidaDb)

	// stop opera to be able to export events
	errCh := startOperaRecording(g.cfg, g.stopAtEpoch)

	// wait for opera recording response
	err, ok := <-errCh
	if ok && err != nil {
		t.Error(err)
		return
	}
	g.log.Noticef("Opera %v - successfully substates for epoch range %d - %d", g.cfg.Db, g.opera.lastEpoch+1, g.stopAtEpoch)

	// reopen aida-db
	g.aidaDb, err = rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		t.Error(err)
		return
	}
	substate.SetSubstateDbBackend(g.aidaDb)

	err = g.opera.getOperaBlockAndEpoch(false)
	if err != nil {
		t.Error(err)
		return
	}

	err = g.Generate()
	if err != nil {
		t.Error(err)
		return
	}

	if strings.Compare(expectedDbHash, hex.EncodeToString(g.dbHash)) != 0 {
		t.Errorf("different hashes!\n expected: %v\n db: %v", expectedDbHash, hex.EncodeToString(g.dbHash))
		return
	}
}
