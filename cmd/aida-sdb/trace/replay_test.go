package trace

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestSdbReplay_AllDbEventsAreIssuedInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[[]operation.Operation](ctrl)
	processor := executor.NewMockProcessor[[]operation.Operation](ctrl)
	ext := executor.NewMockExtension[[]operation.Operation](ctrl)

	cfg := &utils.Config{}
	cfg.DbImpl = "carmen"
	cfg.KeepDb = false

	cfg.First = 0
	cfg.Last = 0

	provider.EXPECT().
		Run(0, 1, gomock.Any()).
		DoAndReturn(func(from int, to int, consumer executor.Consumer[[]operation.Operation]) error {
			for i := from; i < to; i++ {
				consumer(executor.TransactionInfo[[]operation.Operation]{Block: 0, Transaction: 0, Data: testOperationsA})
				consumer(executor.TransactionInfo[[]operation.Operation]{Block: 0, Transaction: 1, Data: testOperationsB})
			}
			return nil
		})

	// All transactions are processed in order
	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[[]operation.Operation](0), gomock.Any()),

		// tx 0
		ext.EXPECT().PreTransaction(executor.AtTransaction[[]operation.Operation](0, 0), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[[]operation.Operation](0, 0), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[[]operation.Operation](0, 0), gomock.Any()),

		// tx 1
		ext.EXPECT().PreTransaction(executor.AtTransaction[[]operation.Operation](0, 1), gomock.Any()),
		processor.EXPECT().Process(executor.AtTransaction[[]operation.Operation](0, 1), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtTransaction[[]operation.Operation](0, 1), gomock.Any()),

		ext.EXPECT().PostRun(executor.AtBlock[[]operation.Operation](1), gomock.Any(), nil),
	)

	if err := replay(cfg, provider, processor, []executor.Extension[[]operation.Operation]{ext}); err != nil {
		t.Errorf("record failed: %v", err)
	}
}
