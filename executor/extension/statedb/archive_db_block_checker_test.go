package statedb

import (
	"os"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
)

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfStateDbDoesNotHaveArchive(t *testing.T) {
	cfg := &utils.Config{}
	cfg.StateDbSrc = t.TempDir()
	err := utils.WriteStateDbInfo(cfg.StateDbSrc, cfg, 0, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info; %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "your state db does not contain archive"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfStateDbDoesNotContainGivenBlockRange(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Last = 11

	cfg.StateDbSrc = t.TempDir()
	cfg.ArchiveMode = true
	err := utils.WriteStateDbInfo(cfg.StateDbSrc, cfg, 10, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info; %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "last block of given archive-db (10) is smaller than given last block (11), please chose last block in range"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfPrimeStateDbDoesNotHaveArchive(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ShadowDb = true
	cfg.StateDbSrc = t.TempDir()
	if err := os.Mkdir(cfg.StateDbSrc+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToPrimaryStateDb, cfg, 0, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "your prime state db does not contain archive"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfPrimeStateDbDoesNotContainGivenBlockRange(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Last = 11

	cfg.StateDbSrc = t.TempDir()
	cfg.ArchiveMode = true
	cfg.ShadowDb = true
	cfg.StateDbSrc = t.TempDir()

	if err := os.Mkdir(cfg.StateDbSrc+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToPrimaryStateDb, cfg, 10, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	if err = os.Mkdir(cfg.StateDbSrc+utils.PathToShadowStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	err = utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToShadowStateDb, cfg, 12, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "last block of given archive-db (10) is smaller than given last block (11), please chose last block in range"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfShadowStateDbDoesNotContainGivenBlockRange(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Last = 11

	cfg.StateDbSrc = t.TempDir()
	cfg.ArchiveMode = true
	cfg.ShadowDb = true
	cfg.StateDbSrc = t.TempDir()

	if err := os.Mkdir(cfg.StateDbSrc+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToPrimaryStateDb, cfg, 12, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	if err = os.Mkdir(cfg.StateDbSrc+utils.PathToShadowStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	err = utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToShadowStateDb, cfg, 10, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "last block of given archive-db (10) is smaller than given last block (11), please chose last block in range"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestArchiveDbBlockChecker_PreRunReturnsErrorIfShadowStateDbDoesNotHaveArchive(t *testing.T) {
	cfg := &utils.Config{}
	cfg.ShadowDb = true
	cfg.ArchiveMode = false
	cfg.StateDbSrc = t.TempDir()
	if err := os.Mkdir(cfg.StateDbSrc+utils.PathToShadowStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToShadowStateDb, cfg, 0, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	cfg.ArchiveMode = true

	if err = os.Mkdir(cfg.StateDbSrc+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err = utils.WriteStateDbInfo(cfg.StateDbSrc+utils.PathToPrimaryStateDb, cfg, 0, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeArchiveBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "your shadow state db does not contain archive"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}
