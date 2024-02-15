package executor

import (
	_ "embed"
	"encoding/json"
	"os"
	"testing"

	statetest "github.com/Fantom-foundation/Aida/ethtest/state_test"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func Test_ethTestProvider_Run(t *testing.T) {
	pathFile := createTestDataFile(t)

	cfg := &utils.Config{
		ArgPath:     pathFile,
		EthTestType: utils.EthStateTests,
	}

	provider := NewEthTestProvider(cfg)

	ctrl := gomock.NewController(t)

	var consumer = NewMockTxConsumer(ctrl)

	gomock.InOrder(
		consumer.EXPECT().Consume(1, 0, gomock.Any()),
	)

	err := provider.Run(0, 0, toSubstateConsumer(consumer))
	if err != nil {
		t.Errorf("Run() error = %v, wantErr %v", err, nil)
	}
}

func createTestDataFile(t *testing.T) string {
	path := t.TempDir()
	pathFile := path + "/test.json"
	stData := statetest.CreateTestData(t)

	jsonData, err := json.Marshal(stData)
	if err != nil {
		t.Errorf("Marshal() error = %v, wantErr %v", err, nil)
	}

	jsonStr := "{ \"test\" : " + string(jsonData) + "}"

	jsonData = []byte(jsonStr)
	// Initialize pathFile
	err = os.WriteFile(pathFile, jsonData, 0644)
	if err != nil {
		t.Errorf("WriteFile() error = %v, wantErr %v", err, nil)
	}
	return pathFile
}
