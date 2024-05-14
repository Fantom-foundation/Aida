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

package profiler

import (
	"net/http"
	"testing"
	"time"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestDiagnosticServer_CollectsProfileDataIfEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	cfg := &utils.Config{}
	cfg.DiagnosticServer = 6060
	ext := makeDiagnosticServer[any](cfg, log)

	// Expect a server info message and a warning on the performance impact.
	log.EXPECT().Infof(gomock.Any(), gomock.Any())
	log.EXPECT().Warning(gomock.Any())

	if err := ext.PreRun(executor.State[any]{}, nil); err != nil {
		t.Fatalf("failed to to run pre-run: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Test that the server is online.
	_, err := http.Get("http://localhost:6060")
	if err != nil {
		t.Errorf("Unable to connect to server: %v", err)
	}
}

func TestDiagnosticServer_NoServerIsHostedWhenDisabled(t *testing.T) {
	cfg := &utils.Config{}
	ext := MakeDiagnosticServer[any](cfg)

	if _, ok := ext.(extension.NilExtension[any]); !ok {
		t.Errorf("profiler is enabled although not set in configuration")
	}
}
