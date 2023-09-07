package extension

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"
)

const exampleHashes = `
0 - 0x0100000000000000000000000000000000000000000000000000000000000000
1 - 0x0102000000000000000000000000000000000000000000000000000000000000
2 - 0x0a0b0c0000000000000000000000000000000000000000000000000000000000
3 - 0x0300000000000000000000000000000000000000000000000000000000000000
6 - 0x0f00000000000000000000000000000000000000000000000000000000000000
`

func TestStateHashValidator_NotActiveIfNoFileIsProvided(t *testing.T) {
	config := &utils.Config{}
	ext := MakeStateHashValidator(config)
	if _, ok := ext.(NilExtension); !ok {
		t.Errorf("extension is active although it should not")
	}
}

func TestStateHashValidator_ActiveIfAFileIsProvided(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	path := t.TempDir() + "/hashes.dat"
	if err := os.WriteFile(path, []byte(exampleHashes), 0600); err != nil {
		t.Fatalf("failed to prepare input file: %v", err)
	}

	gomock.InOrder(
		log.EXPECT().Infof("Loading state root hashes from %v ...\n", path),
		log.EXPECT().Infof("Loaded %d state root hashes from %v\n", 6, path),
		db.EXPECT().GetHash().Return(common.Hash{0x03}),
	)

	config := &utils.Config{}
	config.StateRootFile = path
	config.Last = 5
	ext := makeStateHashValidator(config, log)

	if err := ext.PreRun(executor.State{}); err != nil {
		t.Errorf("failed to initialize extension: %v", err)
	}

	if err := ext.PostBlock(executor.State{Block: 4, State: db}); err != nil {
		t.Errorf("failed to check hash: %v", err)
	}
}

func TestStateHashValidator_InvalidHashIsDetected(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)
	db := state.NewMockStateDB(ctrl)

	path := t.TempDir() + "/hashes.dat"
	if err := os.WriteFile(path, []byte(exampleHashes), 0600); err != nil {
		t.Fatalf("failed to prepare input file: %v", err)
	}

	log.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
	db.EXPECT().GetHash().Return(common.Hash{0x04})

	config := &utils.Config{}
	config.StateRootFile = path
	config.Last = 5
	ext := makeStateHashValidator(config, log)

	if err := ext.PreRun(executor.State{}); err != nil {
		t.Errorf("failed to initialize extension: %v", err)
	}

	if err := ext.PostBlock(executor.State{Block: 4, State: db}); err == nil || !strings.Contains(err.Error(), "unexpected hash for block 4") {
		t.Errorf("failed to detect incorrect hash, err %v", err)
	}
}

func TestStateRootHashes_LoadHashesFailsForNonexistingFile(t *testing.T) {
	_, err := loadStateHashes(t.TempDir()+"/non_existing.dat", 12)
	if err == nil {
		t.Errorf("loading should have failed")
	}
}

func TestStateRootHashes_LoadHashesWorksOnRegularInput(t *testing.T) {
	path := t.TempDir() + "/hashes.dat"
	if err := os.WriteFile(path, []byte(exampleHashes), 0600); err != nil {
		t.Fatalf("failed to prepare input file: %v", err)
	}

	hashes, err := loadStateHashes(path, 3)
	if err != nil {
		t.Fatalf("failed to load hashes: %v", err)
	}
	want := []common.Hash{{1}, {1, 2}, {0xa, 0xb, 0xc}}
	if !reflect.DeepEqual(hashes, want) {
		t.Errorf("failed to load hashes from files\nwanted: %v\ngot: %v", want, hashes)
	}
}

func TestStateRootHashes_SkippedHashesAreFilled(t *testing.T) {
	path := t.TempDir() + "/hashes.dat"
	if err := os.WriteFile(path, []byte(exampleHashes), 0600); err != nil {
		t.Fatalf("failed to prepare input file: %v", err)
	}

	hashes, err := loadStateHashes(path, 7)
	if err != nil {
		t.Fatalf("failed to load hashes: %v", err)
	}
	want := []common.Hash{{1}, {1, 2}, {0xa, 0xb, 0xc}, {3}, {3}, {3}, {0xf}}
	if !reflect.DeepEqual(hashes, want) {
		t.Errorf("failed to load hashes from files\nwanted: %v\ngot: %v", want, hashes)
	}
}

func TestStateRootHashes_InvalidLineFormatIsDetected(t *testing.T) {
	path := t.TempDir() + "/hashes.dat"
	if err := os.WriteFile(path, []byte("0 - 0x00000000 - 12"), 0600); err != nil {
		t.Fatalf("failed to prepare input file: %v", err)
	}

	if _, err := loadStateHashes(path, 10); err == nil || !strings.Contains(err.Error(), "invalid line") {
		t.Errorf("failed to detect invalid line format; err: %v", err)
	}
}

func TestStateRootHashes_InvalidBlockIsDetected(t *testing.T) {
	path := t.TempDir() + "/hashes.dat"
	if err := os.WriteFile(path, []byte("not_a_block - 0x00000000"), 0600); err != nil {
		t.Fatalf("failed to prepare input file: %v", err)
	}

	if _, err := loadStateHashes(path, 10); err == nil || !strings.Contains(err.Error(), "invalid syntax") {
		t.Errorf("failed to detect invalid block number; err: %v", err)
	}
}

func TestStateRootHashes_InvalidHashIsDetected(t *testing.T) {
	path := t.TempDir() + "/hashes.dat"
	if err := os.WriteFile(path, []byte("12 - not_a_hash"), 0600); err != nil {
		t.Fatalf("failed to prepare input file: %v", err)
	}

	if _, err := loadStateHashes(path, 15); err == nil || !strings.Contains(err.Error(), "unable to decode") {
		t.Errorf("failed to detect invalid block hash; err: %v", err)
	}
}

func TestStateRootHashes_OutOfOrderHashesAreDetected(t *testing.T) {
	path := t.TempDir() + "/hashes.dat"
	if err := os.WriteFile(path, []byte("2 - 0x00\n1 - 0x00\n"), 0600); err != nil {
		t.Fatalf("failed to prepare input file: %v", err)
	}

	if _, err := loadStateHashes(path, 15); err == nil || !strings.Contains(err.Error(), "lines in state hash file are not sorted") {
		t.Errorf("failed to detect invalid block hash; err: %v", err)
	}
}
