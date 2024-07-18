// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	_ "embed"
	"encoding/json"
	"os"
	"testing"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func Test_ethTestProvider_Run(t *testing.T) {
	pathFile := createTestDataFile(t)

	cfg := &utils.Config{
		ArgPath: pathFile,
		Forks:   []string{"all"},
	}

	provider := NewEthStateTestProvider(cfg)

	ctrl := gomock.NewController(t)

	var consumer = NewMockTxConsumer(ctrl)

	gomock.InOrder(
		consumer.EXPECT().Consume(2, 0, gomock.Any()),
		consumer.EXPECT().Consume(2, 1, gomock.Any()),
		consumer.EXPECT().Consume(2, 2, gomock.Any()),
		consumer.EXPECT().Consume(2, 3, gomock.Any()),
	)

	err := provider.Run(0, 0, toSubstateConsumer(consumer))
	if err != nil {
		t.Errorf("Run() error = %v, wantErr %v", err, nil)
	}
}

func createTestDataFile(t *testing.T) string {
	path := t.TempDir()
	pathFile := path + "/test.json"
	stData := ethtest.CreateTestStJson(t)

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
