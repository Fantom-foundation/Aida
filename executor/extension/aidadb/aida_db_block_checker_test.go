package aidadb

import (
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
)

func TestAidaDbBlockChecker_PreRunReturnsErrorIfFirstBlockIsNotWithinAidaDb(t *testing.T) {
	cfg := &utils.Config{}
	cfg.First = 0

	ext := makeAidaDbBlockChecker[any](cfg, 1, 2)
	err := ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "first block of given aida-db (1) is larger than given first block (0), please chose first block in range"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestAidaDbBlockChecker_PreRunReturnsErrorIfLastBlockIsNotWithinAidaDb(t *testing.T) {
	cfg := &utils.Config{}
	cfg.First = 1
	cfg.Last = 3

	ext := makeAidaDbBlockChecker[any](cfg, 1, 2)
	err := ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "last block of given aida-db (2) is smaller than given last block (3), please choose last block in range"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}

func TestAidaDbBlockChecker_PreRunReturnsErrorIfSubstateWasNotFound(t *testing.T) {
	cfg := &utils.Config{}
	cfg.First = 1
	cfg.Last = 3

	ext := makeAidaDbBlockChecker[any](cfg, 0, 0)
	err := ext.PreRun(executor.State[any]{}, nil)
	if err == nil {
		t.Fatalf("pre-run must return error")
	}

	wantedErr := "your aida-db does not have substate"

	if strings.Compare(err.Error(), wantedErr) != 0 {
		t.Fatalf("unexpected err\ngot: %v\n want: %v", err, wantedErr)
	}
}
