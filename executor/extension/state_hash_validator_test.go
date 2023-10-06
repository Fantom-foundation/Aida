package extension

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
	"go.uber.org/mock/gomock"
)

const exampleHashA = "0x0100000000000000000000000000000000000000000000000000000000000000"
const exampleHashB = "0x0102000000000000000000000000000000000000000000000000000000000000"
const exampleHashC = "0x0a0b0c0000000000000000000000000000000000000000000000000000000000"
const exampleHashD = "0x0300000000000000000000000000000000000000000000000000000000000000"

func TestStateHashValidator_NotActiveIfNotEnabledInConfig(t *testing.T) {
	config := &utils.Config{}
	config.ValidateStateHashes = false

	ext := MakeStateHashValidator(config)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("extension is active although it should not")
	}
}

func TestStateHashValidator_DoesNotFailIfHashIsNotFoundInAidaDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	hashProvider := utils.NewMockStateHashProvider(ctrl)

	blockNumber := 1

	gomock.InOrder(
		hashProvider.EXPECT().GetStateHash(blockNumber).Return(common.Hash{}, leveldb.ErrNotFound),
		log.EXPECT().Warningf("State hash for block %v is not present in the db", blockNumber),
	)

	config := &utils.Config{}
	ext := makeStateHashValidator(config, log)
	ext.hashProvider = hashProvider

	ctx := &executor.Context{State: db}

	if err := ext.PostBlock(executor.State{Block: blockNumber}, ctx); err != nil {
		t.Errorf("failed to check hash: %v", err)
	}

	if err := ext.PostRun(executor.State{Block: 1}, ctx, nil); err != nil {
		t.Errorf("failed to finish PostRun: %v", err)
	}
}

func TestStateHashValidator_InvalidHashOfLiveDbIsDetected(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	hashProvider := utils.NewMockStateHashProvider(ctrl)

	blockNumber := 1

	config := &utils.Config{}
	ext := makeStateHashValidator(config, log)
	ext.hashProvider = hashProvider

	gomock.InOrder(
		hashProvider.EXPECT().GetStateHash(blockNumber).Return(common.Hash([]byte(exampleHashA)), nil),
		db.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashB))),
	)

	ctx := &executor.Context{State: db}

	if err := ext.PostBlock(executor.State{Block: blockNumber}, ctx); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("unexpected hash for Live block %v", blockNumber)) {
		t.Errorf("failed to detect incorrect hash, err %v", err)
	}
}
func TestStateHashValidator_InvalidHashOfArchiveDbIsDetected(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	hashProvider := utils.NewMockStateHashProvider(ctrl)

	blockNumber := 1

	config := &utils.Config{}
	config.ArchiveMode = true

	ext := makeStateHashValidator(config, log)
	ext.hashProvider = hashProvider

	archive := state.NewMockNonCommittableStateDB(ctrl)

	gomock.InOrder(
		// live state check goes through
		hashProvider.EXPECT().GetStateHash(blockNumber).Return(common.Hash([]byte(exampleHashA)), nil),
		db.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashA))),
		db.EXPECT().GetArchiveBlockHeight().Return(uint64(blockNumber), false, nil),
		db.EXPECT().GetArchiveState(uint64(blockNumber-1)).Return(archive, nil),

		// archive state check fails
		hashProvider.EXPECT().GetStateHash(blockNumber-1).Return(common.Hash([]byte(exampleHashA)), nil),
		archive.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashB))),
		archive.EXPECT().Release(),
	)

	ctx := &executor.Context{State: db}

	if err := ext.PostBlock(executor.State{Block: blockNumber}, ctx); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("unexpected hash for archive block %d", blockNumber-1)) {
		t.Errorf("failed to detect incorrect hash, err %v", err)
	}
}

func TestStateHashValidator_ChecksArchiveHashesOfLaggingArchive(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	hashProvider := utils.NewMockStateHashProvider(ctrl)

	db.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashA)))
	hashProvider.EXPECT().GetStateHash(2).Return(common.Hash([]byte(exampleHashA)), nil)

	archive0 := state.NewMockNonCommittableStateDB(ctrl)
	archive1 := state.NewMockNonCommittableStateDB(ctrl)
	archive2 := state.NewMockNonCommittableStateDB(ctrl)

	gomock.InOrder(
		db.EXPECT().GetArchiveBlockHeight().Return(uint64(0), false, nil),
		db.EXPECT().GetArchiveState(uint64(0)).Return(archive0, nil),
		hashProvider.EXPECT().GetStateHash(0).Return(common.Hash([]byte(exampleHashB)), nil),
		archive0.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashB))),
		archive0.EXPECT().Release(),

		db.EXPECT().GetArchiveBlockHeight().Return(uint64(0), false, nil),
		db.EXPECT().GetArchiveBlockHeight().Return(uint64(0), false, nil),

		db.EXPECT().GetArchiveBlockHeight().Return(uint64(2), false, nil),
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive1, nil),
		hashProvider.EXPECT().GetStateHash(1).Return(common.Hash([]byte(exampleHashC)), nil),
		archive1.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashC))),
		archive1.EXPECT().Release(),
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive2, nil),
		hashProvider.EXPECT().GetStateHash(2).Return(common.Hash([]byte(exampleHashD)), nil),
		archive2.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashA))),
		archive2.EXPECT().Release(),
	)

	config := &utils.Config{}
	config.Last = 5
	config.ArchiveMode = true
	ext := makeStateHashValidator(config, log)
	ext.hashProvider = hashProvider
	context := &executor.Context{State: db}

	// A PostBlock run should check the LiveDB and the ArchiveDB up to block 0.
	if err := ext.PostBlock(executor.State{Block: 2}, context); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// PostRun should finish up checking all remaining archive hashes and detect the error in block 2.
	if err := ext.PostRun(executor.State{Block: 3}, context, nil); err == nil || !strings.Contains(err.Error(), "unexpected hash for archive block 2") {
		t.Errorf("failed to detect incorrect hash, err %v", err)
	}
}

func TestStateHashValidator_ChecksArchiveHashesOfLaggingArchiveDoesNotWaitForNonExistingBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	hashProvider := utils.NewMockStateHashProvider(ctrl)

	db.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashA)))
	hashProvider.EXPECT().GetStateHash(2).Return(common.Hash([]byte(exampleHashA)), nil)

	archive0 := state.NewMockNonCommittableStateDB(ctrl)
	archive1 := state.NewMockNonCommittableStateDB(ctrl)
	archive2 := state.NewMockNonCommittableStateDB(ctrl)

	gomock.InOrder(
		db.EXPECT().GetArchiveBlockHeight().Return(uint64(0), false, nil),
		db.EXPECT().GetArchiveState(uint64(0)).Return(archive0, nil),
		hashProvider.EXPECT().GetStateHash(0).Return(common.Hash([]byte(exampleHashB)), nil),
		archive0.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashB))),
		archive0.EXPECT().Release(),

		db.EXPECT().GetArchiveBlockHeight().Return(uint64(2), false, nil),
		db.EXPECT().GetArchiveState(uint64(1)).Return(archive1, nil),
		hashProvider.EXPECT().GetStateHash(1).Return(common.Hash([]byte(exampleHashC)), nil),
		archive1.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashC))),
		archive1.EXPECT().Release(),
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive2, nil),
		hashProvider.EXPECT().GetStateHash(2).Return(common.Hash([]byte(exampleHashD)), nil),
		archive2.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashD))),
		archive2.EXPECT().Release(),
	)

	config := &utils.Config{}
	config.Last = 5
	config.ArchiveMode = true
	ext := makeStateHashValidator(config, log)
	ext.hashProvider = hashProvider
	context := &executor.Context{State: db}

	// A PostBlock run should check the LiveDB and the ArchiveDB up to block 0.
	if err := ext.PostBlock(executor.State{Block: 2}, context); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// PostRun should finish up checking all remaining archive blocks, even if the
	// there are some blocks missing at the end of the range.
	if err := ext.PostRun(executor.State{Block: 10}, context, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStateHashValidator_ValidatingLaggingArchivesIsSkippedIfRunIsAborted(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)
	hashProvider := utils.NewMockStateHashProvider(ctrl)

	db.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashA)))
	hashProvider.EXPECT().GetStateHash(2).Return(common.Hash([]byte(exampleHashA)), nil)

	archive0 := state.NewMockNonCommittableStateDB(ctrl)

	gomock.InOrder(
		db.EXPECT().GetArchiveBlockHeight().Return(uint64(0), false, nil),
		db.EXPECT().GetArchiveState(uint64(0)).Return(archive0, nil),
		hashProvider.EXPECT().GetStateHash(0).Return(common.Hash([]byte(exampleHashB)), nil),
		archive0.EXPECT().GetHash().Return(common.Hash([]byte(exampleHashB))),
		archive0.EXPECT().Release(),
	)

	config := &utils.Config{}
	config.Last = 5
	config.ArchiveMode = true
	ext := makeStateHashValidator(config, log)
	ext.hashProvider = hashProvider
	context := &executor.Context{State: db}

	// A PostBlock run should check the LiveDB and the ArchiveDB up to block 0.
	if err := ext.PostBlock(executor.State{Block: 2}, context); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// PostRun should finish up checking all remaining archive hashes and detect the error in block 2.
	if err := ext.PostRun(executor.State{Block: 2}, context, fmt.Errorf("dummy")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
