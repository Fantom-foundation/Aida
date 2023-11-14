package statedb

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

func TestStateDbManager_DbClosureWithoutKeepDb(t *testing.T) {
	cfg := &utils.Config{}

	ext := MakeStateDbManager[any](cfg)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	mockStateDB.EXPECT().Close()

	state := executor.State[any]{
		Block: 0,
	}

	ctx := &executor.Context{State: mockStateDB}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestStateDbManager_DbClosureWithKeepDb(t *testing.T) {
	cfg := &utils.Config{}

	tmpDir := t.TempDir()
	cfg.DbTmp = tmpDir
	cfg.DbImpl = "geth"
	cfg.KeepDb = true

	ext := MakeStateDbManager[any](cfg)

	mockCtrl := gomock.NewController(t)
	mockStateDB := state.NewMockStateDB(mockCtrl)

	gomock.InOrder(
		mockStateDB.EXPECT().GetHash(),
		mockStateDB.EXPECT().Close(),
	)

	state := executor.State[any]{
		Block: 0,
	}

	// setting mockStateDb and StateDbPath path for statedb_info.json output
	ctx := &executor.Context{State: mockStateDB, StateDbPath: tmpDir}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}
}

func TestStateDbManager_DoNotKeepDb(t *testing.T) {
	cfg := &utils.Config{}

	tmpDir := t.TempDir()
	cfg.DbTmp = tmpDir
	cfg.DbImpl = "geth"
	cfg.KeepDb = false

	ext := MakeStateDbManager[any](cfg)

	state := executor.State[any]{
		Block: 0,
	}

	ctx := &executor.Context{}

	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	empty, err := IsEmptyDirectory(cfg.DbTmp)
	if err != nil {
		t.Fatalf("failed to check DbTmp; %v", err)
	}
	if !empty {
		t.Fatalf("failed to clean up DbTmp %v after post-run; %v", cfg.DbTmp, err)
	}
}

func TestStateDbManager_KeepDbAndDoesntUnderflowBellowZero(t *testing.T) {
	cfg := &utils.Config{}

	tmpDir := t.TempDir()
	cfg.DbTmp = tmpDir
	cfg.DbImpl = "geth"
	cfg.KeepDb = true

	ext := MakeStateDbManager[any](cfg)

	state := executor.State[any]{
		Block: 0,
	}

	ctx := &executor.Context{}

	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	expectedName := fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, state.Block)
	expectedPath := filepath.Join(cfg.DbTmp, expectedName)

	_, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("failed to create stateDb; %v", err)
	}
}

func TestStateDbManager_StateDbInfoExistence(t *testing.T) {
	cfg := &utils.Config{}

	tmpDir := t.TempDir()
	cfg.DbTmp = tmpDir
	cfg.DbImpl = "geth"
	cfg.KeepDb = true

	ext := MakeStateDbManager[any](cfg)

	state := executor.State[any]{
		Block: 0,
	}

	ctx := &executor.Context{}

	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	expectedName := fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, state.Block)
	expectedPath := filepath.Join(cfg.DbTmp, expectedName)

	filename := filepath.Join(expectedPath, utils.PathToDbInfo)

	_, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("failed to find %v of stateDbInfo; %v", utils.PathToDbInfo, err)
	}
}

func TestStateDbManager_NonExistentStateDbSrc(t *testing.T) {
	cfg := &utils.Config{}

	tmpDir := t.TempDir()
	cfg.DbTmp = tmpDir
	cfg.StateDbSrc = "/non-existant-path/123456789"
	cfg.DbImpl = "geth"
	cfg.KeepDb = true

	ext := MakeStateDbManager[any](cfg)

	state := executor.State[any]{
		Block: 0,
	}

	ctx := &executor.Context{}

	err := ext.PreRun(state, ctx)
	if err == nil {
		t.Fatalf("using nonexistent statedb didn't produce error in pre-run")
	}

	if strings.Compare(err.Error(), fmt.Sprintf("%v does not exist", cfg.StateDbSrc)) != 0 {
		t.Fatalf("unexpected error")
	}
}

func TestStateDbManager_StateDbSrcStateDbIsReadOnly(t *testing.T) {
	cfg := &utils.Config{}

	// create source database
	tmpDir := t.TempDir()
	cfg.DbTmp = tmpDir
	cfg.DbImpl = "geth"
	cfg.KeepDb = true

	ext := MakeStateDbManager[any](cfg)

	state0 := executor.State[any]{
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

	expectedName := fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, state0.Block)
	sourcePath := filepath.Join(cfg.DbTmp, expectedName)

	tmpOutDir := t.TempDir()
	cfg.DbTmp = tmpOutDir
	cfg.StateDbSrc = sourcePath
	cfg.KeepDb = false
	cfg.SrcDbReadonly = true

	ext = MakeStateDbManager[any](cfg)

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
		t.Fatalf("Source StateDb was removed from %v", sourcePath)
	}

	//  check that the readonly database was used instead of using working directory from second run
	empty, err = IsEmptyDirectory(tmpOutDir)
	if err != nil {
		t.Fatalf("failed to retrieve source stateDb; %v", err)
	}
	if !empty {
		t.Fatalf("Source StateDb was removed from %v", sourcePath)
	}
}

func TestStateDbManager_UsingExistingSourceDb(t *testing.T) {
	cfg := &utils.Config{}

	// create source database
	tmpDir := t.TempDir()
	cfg.DbTmp = tmpDir
	cfg.DbImpl = "geth"
	cfg.KeepDb = true

	ext := MakeStateDbManager[any](cfg)

	state0 := executor.State[any]{
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

	expectedName := fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, state0.Block)
	sourcePath := filepath.Join(cfg.DbTmp, expectedName)

	tmpOutDir := t.TempDir()
	cfg.DbTmp = tmpOutDir
	cfg.StateDbSrc = sourcePath

	ext = MakeStateDbManager[any](cfg)

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
		t.Fatalf("Source StateDb was removed from %v", sourcePath)
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

func TestStateDbManager_StateDbBlockNumberDecrements(t *testing.T) {
	cfg := &utils.Config{}

	tmpDir := t.TempDir()
	cfg.DbTmp = tmpDir
	cfg.DbImpl = "geth"
	cfg.KeepDb = true

	ext := MakeStateDbManager[any](cfg)

	state := executor.State[any]{
		Block: 10,
	}

	ctx := &executor.Context{}

	if err := ext.PreRun(state, ctx); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	if err := ext.PostRun(state, ctx, nil); err != nil {
		t.Fatalf("failed to to run post-run: %v", err)
	}

	expectedName := fmt.Sprintf("state_db_%v_%v", cfg.DbImpl, state.Block-1)
	expectedPath := filepath.Join(cfg.DbTmp, expectedName)

	_, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("failed to create stateDb; %v", err)
	}
}
