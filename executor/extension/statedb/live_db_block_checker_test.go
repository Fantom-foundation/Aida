package statedb

import (
	"os"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
)

func TestLiveDbBlockChecker_PreRunReturnsErrorIfStateDbLastBlockIsTooSmall(t *testing.T) {
	cfg := &utils.Config{}
	cfg.First = 15
	cfg.IsExistingStateDb = true

	cfg.PathToStateDb = t.TempDir()
	err := utils.WriteStateDbInfo(cfg.PathToStateDb, cfg, 10, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info; %v", err)
	}

	ext := MakeLiveDbBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "if using existing live-db with vm-sdb first block needs to be last block of live-db + 1, in your case 11"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestLiveDbBlockChecker_PreRunReturnsErrorIfShadowDbLastBlockIsTooSmall(t *testing.T) {
	cfg := &utils.Config{}
	cfg.First = 15
	cfg.IsExistingStateDb = true
	cfg.ShadowDb = true

	cfg.PathToStateDb = t.TempDir()
	if err := os.Mkdir(cfg.PathToStateDb+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.PathToStateDb+utils.PathToPrimaryStateDb, cfg, 10, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	if err = os.Mkdir(cfg.PathToStateDb+utils.PathToShadowStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	err = utils.WriteStateDbInfo(cfg.PathToStateDb+utils.PathToShadowStateDb, cfg, 10, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeLiveDbBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "if using existing live-db with vm-sdb first block needs to be last block of live-db + 1, in your case 11"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestShadowDbBlockChecker_PreRunReturnsErrorIfPrimeAndShadowDbHaveDifferentLastBlock(t *testing.T) {
	cfg := &utils.Config{}
	cfg.First = 15
	cfg.IsExistingStateDb = true
	cfg.ShadowDb = true

	cfg.PathToStateDb = t.TempDir()
	if err := os.Mkdir(cfg.PathToStateDb+utils.PathToPrimaryStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	err := utils.WriteStateDbInfo(cfg.PathToStateDb+utils.PathToPrimaryStateDb, cfg, 11, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	if err = os.Mkdir(cfg.PathToStateDb+utils.PathToShadowStateDb, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	err = utils.WriteStateDbInfo(cfg.PathToStateDb+utils.PathToShadowStateDb, cfg, 10, common.Hash{})
	if err != nil {
		t.Fatalf("cannot create testing state db info %v", err)
	}

	ext := MakeLiveDbBlockChecker[any](cfg)
	err = ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "shadow (11) and prime (10) state dbs have different last block"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}
