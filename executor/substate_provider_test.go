package executor

//go:generate mockgen -source substate_provider_test.go -destination substate_provider_test_mocks.go -package executor

import (
	"errors"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
	"github.com/Fantom-foundation/Substate/substate"
	"go.uber.org/mock/gomock"
)

func TestSubstateProvider_IterateOverExistingDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some substate data.
	path := t.TempDir()
	if err := createSubstateDb(t, path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the substate data for reading.
	provider := openSubstateDb(path, t)
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(10, 7, gomock.Any()),
		consumer.EXPECT().Consume(10, 9, gomock.Any()),
		consumer.EXPECT().Consume(12, 5, gomock.Any()),
	)

	if err := provider.Run(0, 20, toSubstateConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestSubstateProvider_LowerBoundIsInclusive(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some substate data.
	path := t.TempDir()
	if err := createSubstateDb(t, path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the substate data for reading.
	provider := openSubstateDb(path, nil)
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(10, 7, gomock.Any()),
		consumer.EXPECT().Consume(10, 9, gomock.Any()),
		consumer.EXPECT().Consume(12, 5, gomock.Any()),
	)

	if err := provider.Run(10, 20, toSubstateConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestSubstateProvider_UpperBoundIsExclusive(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some substate data.
	path := t.TempDir()
	if err := createSubstateDb(t, path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the substate data for reading.
	provider := openSubstateDb(path, nil)
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(10, 7, gomock.Any()),
		consumer.EXPECT().Consume(10, 9, gomock.Any()),
	)

	if err := provider.Run(10, 12, toSubstateConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestSubstateProvider_RangeCanBeEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some substate data.
	path := t.TempDir()
	if err := createSubstateDb(t, path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the substate data for reading.
	provider := openSubstateDb(path, nil)
	defer provider.Close()

	if err := provider.Run(5, 10, toSubstateConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestSubstateProvider_IterationCanBeAbortedByConsumer(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some substate data.
	path := t.TempDir()
	if err := createSubstateDb(t, path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the substate data for reading.
	provider := openSubstateDb(path, nil)
	defer provider.Close()

	stop := errors.New("stop!")
	gomock.InOrder(
		consumer.EXPECT().Consume(10, 7, gomock.Any()),
		consumer.EXPECT().Consume(10, 9, gomock.Any()).Return(stop),
	)

	if got, want := provider.Run(10, 20, toSubstateConsumer(consumer)), stop; !errors.Is(got, want) {
		t.Errorf("provider run did not finish with expected exception, wanted %d, got %d", want, got)
	}
}

func openSubstateDb(path string, t *testing.T) Provider[txcontext.TxContext] {
	cfg := utils.Config{}
	cfg.AidaDb = path
	cfg.Workers = 1
	aidaDb, err := db.NewDefaultBaseDB(path)
	if err != nil {
		t.Fatal(err)
	}
	return OpenSubstateProvider(&cfg, nil, aidaDb)
}

func createSubstateDb(t *testing.T, path string) error {
	sdb, err := db.NewDefaultSubstateDB(path)
	if err != nil {
		t.Fatal(err)
	}
	state := substate.Substate{
		Block:       10,
		Transaction: 7,
		Env:         &substate.Env{},
		Message: &substate.Message{
			Value: big.NewInt(12),
		},
		InputSubstate:  substate.WorldState{},
		OutputSubstate: substate.WorldState{},
		Result:         &substate.Result{},
	}

	err = sdb.PutSubstate(&state)
	if err != nil {
		t.Fatal(err)
	}

	state.Block = 10
	state.Transaction = 9
	err = sdb.PutSubstate(&state)
	if err != nil {
		t.Fatal(err)
	}

	state.Block = 12
	state.Transaction = 5
	err = sdb.PutSubstate(&state)
	if err != nil {
		t.Fatal(err)
	}

	sdb.Close()
	return nil
}
