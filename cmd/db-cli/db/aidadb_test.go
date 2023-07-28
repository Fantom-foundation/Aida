package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/op/go-logging"
)

const (
	pathToAidaTestDb   = "/var/opera/Aida/testnet-data/aida-db"
	pathToGenesis      = "/var/opera/Aida/testnet-data/genesis/testnet-2458-pruned-mpt.g"
	pathToOperaTestDb  = "/var/opera/Aida/testnet-data/opera-db"
	pathToCloneTestDb  = "/var/opera/Aida/testnet-data/clone-db"
	pathToPatchTestDb  = "/var/opera/Aida/testnet-data/patch"
	pathToMergedTestDb = "/var/opera/Aida/testnet-data/merged-db"

	firstGenAndMergeDbBlock = 479327
	lastGenAndMergeDbBlock  = 481383
	firstCloneDbBlock       = firstGenAndMergeDbBlock
	lastCloneDbBlock        = 480524
	firstPatchDbBlock       = lastCloneDbBlock + 1
	lastPatchDbBlock        = lastGenAndMergeDbBlock

	firstGenAndMergeDbEpoch = 2458
	lastGenAndMergeDbEpoch  = 2459
	firstCloneDbEpoch       = firstGenAndMergeDbEpoch
	lastCloneDbEpoch        = 2458
	firstPatchDbEpoch       = 2459
	lastPatchDbEpoch        = 2459
)

// TestAidaDb by creating one epoch large AidaDb and OperaDb.
// Testnet genesis file is needed for this test.
func TestAidaDb(t *testing.T) {
	var err error

	ctx, cfg := utils.CreateTestEnvironment()

	cfg.AidaDb = pathToAidaTestDb
	cfg.Db = pathToOperaTestDb
	cfg.Genesis = pathToGenesis

	g, err := newGenerator(ctx, cfg)
	if err != nil {
		t.Error(err)
		return
	}

	defer func(log *logging.Logger) {
		if cfg.WorldStateDb != "" {
			err = os.RemoveAll(cfg.WorldStateDb)
			if err != nil {
				log.Criticalf("can't remove temporary folder: %v; %v", cfg.WorldStateDb, err)
			}
		}

		// remove every db after test is done
		deleteAllTestDb(log)
	}(g.log)

	// test autogen by generating 2 epoch large db
	err = testAutogen(cfg, g)
	if err != nil {
		t.Error(err)
		return
	}

	err = testClone(cfg)
	if err != nil {
		t.Error(err)
		return
	}

	err = testMerge(cfg)
	if err != nil {
		t.Error(err)
		return
	}

}

// checkTestMetadata inside given db whether they match with expected gen. range
func checkTestMetadata(db ethdb.Database, typ utils.AidaDbType) error {
	// todo check dbhash

	md := utils.NewAidaDbMetadata(db, "DEBUG")
	fb := md.GetFirstBlock()
	fe := md.GetFirstEpoch()
	lb := md.GetLastBlock()
	le := md.GetLastEpoch()

	var (
		efb, efe, elb, ele uint64
		dbType             string
	)

	switch typ {
	case utils.GenType:
		// gen and merged db are expected to have same range
		fallthrough
	case utils.CloneType:
		efb = firstCloneDbBlock
		efe = firstCloneDbEpoch
		elb = lastCloneDbBlock
		ele = lastCloneDbEpoch
		dbType = "clone"
	case utils.PatchType:
		efb = firstPatchDbBlock
		efe = firstPatchDbEpoch
		elb = lastPatchDbBlock
		ele = lastPatchDbEpoch
		dbType = "patch"
	default:
		efb = firstGenAndMergeDbBlock
		efe = firstGenAndMergeDbEpoch
		elb = lastGenAndMergeDbBlock
		ele = lastGenAndMergeDbEpoch
		dbType = "gen/merge"

	}

	if fb != efb {
		return fmt.Errorf("wrong first %v block; expected: %v db: %v", dbType, efb, fb)
	}

	if fe != efe {
		return fmt.Errorf("wrong first %v epoch; expected: %v db: %v", dbType, efe, fe)
	}

	if lb != elb {
		return fmt.Errorf("wrong last %v block; expected: %v db: %v", dbType, elb, lb)
	}

	if le != ele {
		return fmt.Errorf("wrong last %v epoch; expected: %v db: %v", dbType, ele, le)
	}

	return nil
}

// testMerge merges cloneDb and patchDb created by these tests which results in same db as the source (generatedDb)
func testMerge(cfg *utils.Config) error {
	cfg.AidaDb = pathToCloneTestDb

	targetDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return fmt.Errorf("cannot open db; %v", err)
	}

	var (
		dbs         []ethdb.Database
		md          *utils.AidaDbMetadata
		sourcePaths = []string{pathToPatchTestDb}
	)

	dbs, err = openSourceDatabases(sourcePaths)
	if err != nil {
		return err
	}
	md, err = utils.ProcessMergeMetadata(cfg, targetDb, dbs, sourcePaths)
	if err != nil {
		return err
	}

	targetDb = md.Db

	for _, db := range dbs {
		MustCloseDB(db)
	}

	dbs, err = openSourceDatabases(sourcePaths)
	if err != nil {
		return err
	}

	m := newMerger(cfg, targetDb, dbs, sourcePaths, md)

	if err = m.merge(); err != nil {
		return err
	}

	m.closeSourceDbs()

	err = m.finishMerge()
	if err != nil {
		return err
	}

	err = checkTestMetadata(m.targetDb, utils.NoType)
	if err != nil {
		return err
	}

	return nil
}

// testAutogen by generating small db (2 epoch large) that is used in later stages for testing cloning as a source db.
func testAutogen(cfg *utils.Config, g *generator) error {
	var err error

	err = g.opera.init()
	if err != nil {
		return err
	}

	err = g.calculatePatchEnd()
	if err != nil {
		return err
	}

	// set range only on one epoch
	g.stopAtEpoch = g.opera.lastEpoch + 2

	g.log.Noticef("Starting substate generation %d - %d", g.opera.lastEpoch+1, g.stopAtEpoch)

	MustCloseDB(g.aidaDb)

	// stop opera to be able to export events
	errCh := startOperaRecording(g.cfg, g.stopAtEpoch)

	// wait for opera recording response
	err, ok := <-errCh
	if ok && err != nil {
		return err
	}
	g.log.Noticef("Opera %v - successfully substates for epoch range %d - %d", g.cfg.Db, g.opera.lastEpoch+1, g.stopAtEpoch)

	// reopen aida-db
	g.aidaDb, err = rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return err
	}
	substate.SetSubstateDbBackend(g.aidaDb)

	err = g.opera.getOperaBlockAndEpoch(false)
	if err != nil {
		return err
	}

	err = g.Generate()
	if err != nil {
		return err
	}

	err = checkTestMetadata(g.aidaDb, utils.GenType)
	if err != nil {
		return err
	}

	return nil
}

// testClone creates cloneDb for range of testing AidaDb - 10
func testClone(cfg *utils.Config) error {
	// set testing variables
	cfg.TargetDb = pathToCloneTestDb
	cfg.First = firstCloneDbBlock
	cfg.Last = lastCloneDbBlock

	aidaDb, targetDb, err := openCloningDbs(cfg.AidaDb, cfg.TargetDb)
	if err != nil {
		return err
	}

	err = clone(cfg, aidaDb, targetDb, utils.CloneType, false)
	if err != nil {
		return err
	}

	err = checkTestMetadata(aidaDb, utils.CloneType)
	if err != nil {
		return err
	}

	MustCloseDB(targetDb)
	MustCloseDB(aidaDb)

	// set the block range, so it aligns with clone db, so we can use these dbs for testing merge
	cfg.TargetDb = pathToPatchTestDb
	cfg.First = firstPatchDbBlock
	cfg.Last = lastPatchDbBlock

	aidaDb, targetDb, err = openCloningDbs(cfg.AidaDb, cfg.TargetDb)
	if err != nil {
		return err
	}

	err = CreatePatchClone(cfg, aidaDb, targetDb, firstPatchDbEpoch, lastPatchDbEpoch, false)
	if err != nil {
		return err
	}

	err = checkTestMetadata(aidaDb, utils.PatchType)
	if err != nil {
		return err
	}

	MustCloseDB(aidaDb)
	MustCloseDB(targetDb)

	return nil
}

// deleteAllTestDb after tests are complete. This is called even though any test fails.
func deleteAllTestDb(log *logging.Logger) {
	var err error
	err = os.RemoveAll(pathToAidaTestDb)
	if err != nil {
		log.Error(err)
	}

	err = os.RemoveAll(pathToOperaTestDb)
	if err != nil {
		log.Error(err)
	}

	err = os.RemoveAll(pathToCloneTestDb)
	if err != nil {
		log.Error(err)
	}

	err = os.RemoveAll(pathToPatchTestDb)
	if err != nil {
		log.Error(err)
	}

	err = os.RemoveAll(pathToMergedTestDb)
	if err != nil {
		log.Error(err)
	}
}
