package extension

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestDbManager_DoNotKeepDb(t *testing.T) {
	config := &utils.Config{}

	tmpDir := t.TempDir()
	config.StateDbSrc = tmpDir
	config.KeepDb = false

	ext := MakeDbManager(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	ext.PostRun(executor.State{
		Block: 0,
		State: mockStateDB,
	}, nil)
}

func TestDbManager_KeepDb(t *testing.T) {
	config := &utils.Config{}

	// need two separate tempDirs to be able to move db to new location
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)
	config.StateDbSrc = tmpDir

	tmpOutDir := t.TempDir()
	defer os.RemoveAll(tmpOutDir)
	config.DbTmp = tmpOutDir
	config.DbImpl = "geth"
	config.KeepDb = true

	ext := MakeDbManager(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().Commit(true),
	)

	state := executor.State{
		Block: 0,
		State: mockStateDB,
	}

	ext.PostRun(state, nil)

	expectedName := fmt.Sprintf("state_db_%v_%v", config.DbImpl, state.Block)
	expectedPath := filepath.Join(config.DbTmp, expectedName)

	_, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("failed to create stateDb; %v", err)
	}
}
