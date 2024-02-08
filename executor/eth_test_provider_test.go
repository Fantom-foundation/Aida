package executor

import (
	_ "embed"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func Test_ethTestProvider_Run(t *testing.T) {
	pathFile := ethtest.CreateTestDataFile(t)

	cfg := &utils.Config{
		ArgPath: pathFile,
	}

	provider := NewEthStateTestProvider(cfg)

	ctrl := gomock.NewController(t)

	var consumer = NewMockTxConsumer(ctrl)

	gomock.InOrder(
		consumer.EXPECT().Consume(1, 0, gomock.Any()),
		//consumer.EXPECT().Consume(1, 1, gomock.Any()),
		//consumer.EXPECT().Consume(1, 2, gomock.Any()),
	)

	err := provider.Run(0, 0, toSubstateConsumer(consumer))
	if err != nil {
		t.Errorf("Run() error = %v, wantErr %v", err, nil)
	}
}
