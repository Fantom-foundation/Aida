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
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"go.uber.org/mock/gomock"
)

func TestVirtualMachineStatisticsPrinter_WorksWithDefaultSetup(t *testing.T) {
	cfg := utils.Config{}
	ext := MakeVirtualMachineStatisticsPrinter[any](&cfg)
	ext.PostRun(executor.State[any]{}, nil, nil)
}

func TestVirtualMachineStatisticsPrinter_TriggersStatPrintingAtEndOfRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	interpreter := tosca.NewMockProfilingInterpreter(ctrl)
	tosca.RegisterInterpreter("test-vm", interpreter)

	interpreter.EXPECT().DumpProfile()

	cfg := utils.Config{}
	cfg.VmImpl = "test-vm"
	ext := MakeVirtualMachineStatisticsPrinter[any](&cfg)

	ext.PostRun(executor.State[any]{}, nil, nil)
}
