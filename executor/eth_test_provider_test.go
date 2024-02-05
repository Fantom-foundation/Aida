package executor

import (
	"os"

	_ "embed"
	"testing"

	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

//go:embed eth_test_provider_test_sample.json
var sampleTestData []byte

func Test_ethTestProvider_Run(t *testing.T) {
	path := t.TempDir()
	pathFile := path + "/test.json"

	// Initialize pathFile
	err := os.WriteFile(pathFile, sampleTestData, 0644)
	if err != nil {
		t.Errorf("WriteFile() error = %v, wantErr %v", err, nil)
	}

	cfg := &utils.Config{
		ArgPath: pathFile,
	}

	provider := NewEthStateTestProvider(cfg)

	ctrl := gomock.NewController(t)

	var consumer = NewMockTxConsumer(ctrl)

	gomock.InOrder(
		consumer.EXPECT().Consume(1, 0, gomock.Any()),
		consumer.EXPECT().Consume(1, 1, gomock.Any()),
		consumer.EXPECT().Consume(1, 2, gomock.Any()),
	)

	err = provider.Run(0, 0, toSubstateConsumer(consumer))
	if err != nil {
		t.Errorf("Run() error = %v, wantErr %v", err, nil)
	}
}
