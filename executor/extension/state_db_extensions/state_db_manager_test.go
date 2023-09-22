package state_db_extensions

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestStateDbManager_DbClosureWithoutKeepDb(t *testing.T) {
	config := &utils.Config{}

	ext := MakeStateDbManager(config)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	mockStateDB.EXPECT().Close()

	state := executor.State{
		Block: 0,
	}

	ctx := &executor.Context{State: mockStateDB}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestStateDbManager_DbClosureWithKeepDb(t *testing.T) {
	config := &utils.Config{}

	tmpDir := t.TempDir()
	config.DbTmp = tmpDir
	config.DbImpl = "geth"
	config.KeepDb = true

	ext := MakeStateDbManager(config)

	// setting dummy StateDbPath path for statedb_info.json output
	ext.stateDbPath = tmpDir

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().GetHash(),
		mockStateDB.EXPECT().Close(),
	)

	state := executor.State{
		Block: 0,
	}

	ctx := &executor.Context{State: mockStateDB}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestStateDbManager_DoNotKeepDb(t *testing.T) {
	config := &utils.Config{}

	tmpDir := t.TempDir()
	config.DbTmp = tmpDir
	config.DbImpl = "geth"
	config.KeepDb = false

	ext := MakeStateDbManager(config)

	state := executor.State{
		Block: 0,
	}

	ctx := &executor.Context{}

	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	empty, err := IsEmptyDirectory(config.DbTmp)
	if err != nil {
		t.Fatalf("failed to check DbTmp; %v", err)
	}
	if !empty {
		t.Fatalf("failed to clean up DbTmp %v after post-run; %v", config.DbTmp, err)
	}
}

func TestStateDbManager_KeepDb(t *testing.T) {
	config := &utils.Config{}

	tmpDir := t.TempDir()
	config.DbTmp = tmpDir
	config.DbImpl = "geth"
	config.KeepDb = true

	ext := MakeStateDbManager(config)

	state := executor.State{
		Block: 0,
	}

	ctx := &executor.Context{}

	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	expectedName := fmt.Sprintf("state_db_%v_%v", config.DbImpl, state.Block)
	expectedPath := filepath.Join(config.DbTmp, expectedName)

	_, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("failed to create stateDb; %v", err)
	}
}

func TestStateDbManager_StateDbInfoExistence(t *testing.T) {
	config := &utils.Config{}

	tmpDir := t.TempDir()
	config.DbTmp = tmpDir
	config.DbImpl = "geth"
	config.KeepDb = true

	ext := MakeStateDbManager(config)

	state := executor.State{
		Block: 0,
	}

	ctx := &executor.Context{}

	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	expectedName := fmt.Sprintf("state_db_%v_%v", config.DbImpl, state.Block)
	expectedPath := filepath.Join(config.DbTmp, expectedName)

	filename := filepath.Join(expectedPath, utils.PathToDbInfo)

	_, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("failed to find %v of stateDbInfo; %v", utils.PathToDbInfo, err)
	}
}

func TestStateDbManager_UsingExistingSourceDb(t *testing.T) {
	config := &utils.Config{}

	// create source database
	tmpDir := t.TempDir()
	config.DbTmp = tmpDir
	config.DbImpl = "geth"
	config.KeepDb = true

	ext := MakeStateDbManager(config)

	state0 := executor.State{
		Block: 0,
	}

	ctx := &executor.Context{}

	if err := ext.PreRun(state0, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	// insert random data into the source
	// then the second stateDb hash can be confirmed as correct copy of the source
	insertRandomDataIntoStateDb(t, ctx)

	expectedHash := ctx.State.GetHash()

	if err := ext.PostRun(state0, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	// create database from source

	expectedName := fmt.Sprintf("state_db_%v_%v", config.DbImpl, state0.Block)
	sourcePath := filepath.Join(config.DbTmp, expectedName)

	tmpOutDir := t.TempDir()
	config.DbTmp = tmpOutDir
	config.StateDbSrc = sourcePath
	config.CopySrcDb = true

	ext = MakeStateDbManager(config)

	ctx = &executor.Context{}

	if err := ext.PreRun(state0, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	currentHash := ctx.State.GetHash()

	if currentHash != expectedHash {
		t.Fatalf("stateDB created from existing source stateDB had incorrect hash; got: %v expected: %v", currentHash, expectedHash)
	}

	if err := ext.PostRun(state0, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	//  check original source stateDb, that it wasn't deleted
	empty, err := IsEmptyDirectory(sourcePath)
	if err != nil {
		t.Fatalf("failed to retrieve source stateDb; %v", err)
	}
	if empty {
		t.Fatalf("Source StateDb was removed from %v; %v", sourcePath, err)
	}
}

func insertRandomDataIntoStateDb(t *testing.T, ctx *executor.Context) {
	addr := common.BytesToAddress(state.MakeRandomByteSlice(t, 40))

	// get randomized balance
	additionBase := state.GetRandom(1, 1000*5000)
	addition := big.NewInt(int64(additionBase))

	ctx.State.CreateAccount(addr)
	ctx.State.AddBalance(addr, addition)
}

func IsEmptyDirectory(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
