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
	"testing"

	"github.com/Fantom-foundation/Aida/tracer/context"
	"github.com/Fantom-foundation/Aida/tracer/operation"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestOperationProvider_OpeningANonExistingTraceFilesResultsInAnError(t *testing.T) {
	cfg := utils.Config{}
	cfg.TraceFile = t.TempDir()

	_, err := OpenOperations(&cfg)
	if err == nil {
		t.Errorf("attempting to open a non-existing trace file must fail")
	}
}

func TestOperationProvider_OperationsAreUnitedCorrectly(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockOperationConsumer(ctrl)

	cfg := &utils.Config{}
	cfg.First = 1
	cfg.Last = 3

	cfg.TraceFile = t.TempDir() + "file"
	rCtx, err := context.NewRecord(cfg.TraceFile, 1)
	if err != nil {
		t.Fatal(err)
	}

	operation.WriteOp(rCtx, operation.NewBeginBlock(1))
	operation.WriteOp(rCtx, operation.NewBeginTransaction(0))
	operation.WriteOp(rCtx, operation.NewEndTransaction())
	operation.WriteOp(rCtx, operation.NewBeginTransaction(1))
	operation.WriteOp(rCtx, operation.NewEndTransaction())
	operation.WriteOp(rCtx, operation.NewEndBlock())
	operation.WriteOp(rCtx, operation.NewBeginBlock(2))
	operation.WriteOp(rCtx, operation.NewBeginTransaction(0))
	operation.WriteOp(rCtx, operation.NewEndTransaction())
	operation.WriteOp(rCtx, operation.NewBeginTransaction(1))
	operation.WriteOp(rCtx, operation.NewEndTransaction())
	operation.WriteOp(rCtx, operation.NewEndBlock())

	rCtx.Close()

	provider, err := OpenOperations(cfg)
	if err != nil {
		t.Fatalf("failed to open trace file: %v", err)
	}
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(1, 0, gomock.Any()),
		consumer.EXPECT().Consume(1, 1, gomock.Any()),
		consumer.EXPECT().Consume(2, 0, gomock.Any()),
		consumer.EXPECT().Consume(2, 1, gomock.Any()),
	)

	if err := provider.Run(1, 3, toOperationConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestOperationProvider_BeginBlockSetsBlockNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockOperationConsumer(ctrl)

	cfg := &utils.Config{}
	cfg.First = 0
	cfg.Last = 99

	cfg.TraceFile = t.TempDir() + "file"
	rCtx, err := context.NewRecord(cfg.TraceFile, 1)
	if err != nil {
		t.Fatal(err)
	}

	operation.WriteOp(rCtx, operation.NewBeginBlock(99))
	operation.WriteOp(rCtx, operation.NewEndTransaction())
	operation.WriteOp(rCtx, operation.NewEndBlock())
	rCtx.Close()

	provider, err := OpenOperations(cfg)
	if err != nil {
		t.Fatalf("failed to open trace file: %v", err)
	}
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(99, gomock.Any(), gomock.Any()),
	)

	// even tho we start at block 0 the block number gets changed
	if err := provider.Run(0, 100, toOperationConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}

func TestOperationProvider_BeginTransactionSetsTransactionNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockOperationConsumer(ctrl)

	cfg := &utils.Config{}
	cfg.First = 0
	cfg.Last = 99

	cfg.TraceFile = t.TempDir() + "file"
	rCtx, err := context.NewRecord(cfg.TraceFile, 1)
	if err != nil {
		t.Fatal(err)
	}

	operation.WriteOp(rCtx, operation.NewBeginTransaction(99))
	operation.WriteOp(rCtx, operation.NewEndTransaction())
	operation.WriteOp(rCtx, operation.NewEndBlock())
	rCtx.Close()

	provider, err := OpenOperations(cfg)
	if err != nil {
		t.Fatalf("failed to open trace file: %v", err)
	}
	defer provider.Close()

	gomock.InOrder(
		consumer.EXPECT().Consume(gomock.Any(), 99, gomock.Any()),
	)

	// even tho we start at block 0 the block number gets changed
	if err := provider.Run(0, 100, toOperationConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through states: %v", err)
	}
}
