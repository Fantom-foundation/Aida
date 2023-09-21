package action_provider

//go:generate mockgen -source substate_provider_test.go -destination substate_provider_test_mocks.go -package executor

import (
	"errors"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	substate "github.com/Fantom-foundation/Substate"
	"go.uber.org/mock/gomock"
)

func TestSubstateProvider_OpeningANonExistingDbResultsInAnError(t *testing.T) {
	config := utils.Config{}
	config.AidaDb = t.TempDir()
	// Important: the following code does not panic.
	_, err := OpenSubstateDb(&config, nil)
	if err == nil {
		t.Errorf("attempting to open a non-existing provider DB should fail")
	}
}

func TestSubstateProvider_IterateOverExistingDb(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some provider data.
	path := t.TempDir()
	if err := createSubstateDb(path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the provider data for reading.
	provider, err := openSubstateDb(path)
	if err != nil {
		t.Fatalf("failed to open provider DB: %v", err)
	}
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(10, 7, gomock.Any()),
		consumer.EXPECT().Consume(10, 9, gomock.Any()),
		consumer.EXPECT().Consume(12, 5, gomock.Any()),
	)

	if err := provider.Run(0, 20, toConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestSubstateProvider_LowerBoundIsInclusive(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some provider data.
	path := t.TempDir()
	if err := createSubstateDb(path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the provider data for reading.
	provider, err := openSubstateDb(path)
	if err != nil {
		t.Fatalf("failed to open provider DB: %v", err)
	}
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(10, 7, gomock.Any()),
		consumer.EXPECT().Consume(10, 9, gomock.Any()),
		consumer.EXPECT().Consume(12, 5, gomock.Any()),
	)

	if err := provider.Run(10, 20, toConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestSubstateProvider_UpperBoundIsExclusive(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some provider data.
	path := t.TempDir()
	if err := createSubstateDb(path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the provider data for reading.
	provider, err := openSubstateDb(path)
	if err != nil {
		t.Fatalf("failed to open provider DB: %v", err)
	}
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(10, 7, gomock.Any()),
		consumer.EXPECT().Consume(10, 9, gomock.Any()),
	)

	if err := provider.Run(10, 12, toConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestSubstateProvider_RangeCanBeEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some provider data.
	path := t.TempDir()
	if err := createSubstateDb(path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the provider data for reading.
	provider, err := openSubstateDb(path)
	if err != nil {
		t.Fatalf("failed to open provider DB: %v", err)
	}
	defer provider.Close()

	if err := provider.Run(5, 10, toConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestSubstateProvider_IterationCanBeAbortedByConsumer(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockTxConsumer(ctrl)

	// Prepare a directory containing some provider data.
	path := t.TempDir()
	if err := createSubstateDb(path); err != nil {
		t.Fatalf("failed to setup test DB: %v", err)
	}

	// Open the provider data for reading.
	provider, err := openSubstateDb(path)
	if err != nil {
		t.Fatalf("failed to open provider DB: %v", err)
	}
	defer provider.Close()

	stop := errors.New("stop!")
	gomock.InOrder(
		consumer.EXPECT().Consume(10, 7, gomock.Any()),
		consumer.EXPECT().Consume(10, 9, gomock.Any()).Return(stop),
	)

	if got, want := provider.Run(10, 20, toConsumer(consumer)), stop; !errors.Is(got, want) {
		t.Errorf("provider run did not finish with expected exception, wanted %d, got %d", want, got)
	}
}

type TxConsumer interface {
	Consume(block int, transaction int, substate *substate.Substate) error
}

func toConsumer(c TxConsumer) Consumer {
	return func(info TransactionInfo, _ operation.Operation) error {
		return c.Consume(info.Block, info.Transaction, info.Substate)
	}
}

func openSubstateDb(path string) (ActionProvider, error) {
	config := utils.Config{}
	config.AidaDb = path
	config.Workers = 1
	return OpenSubstateDb(&config, nil)
}

func createSubstateDb(path string) error {
	substate.SetSubstateDb(path)
	substate.OpenSubstateDB()

	state := substate.Substate{
		Env: &substate.SubstateEnv{},
		Message: &substate.SubstateMessage{
			Value: big.NewInt(12),
		},
		InputAlloc:  substate.SubstateAlloc{},
		OutputAlloc: substate.SubstateAlloc{},
		Result:      &substate.SubstateResult{},
	}

	substate.PutSubstate(10, 7, &state)
	substate.PutSubstate(10, 9, &state)
	substate.PutSubstate(12, 5, &state)

	substate.CloseSubstateDB()
	return nil
}
