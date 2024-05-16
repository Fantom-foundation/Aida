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

package main

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/state"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/mock/gomock"
)

var testingAddress = "0x0000000000000000000000000000000000000000"

func TestRpc_AllDbEventsAreIssuedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveOne := state.NewMockNonCommittableStateDB(ctrl)
	archiveTwo := state.NewMockNonCommittableStateDB(ctrl)
	archiveThree := state.NewMockNonCommittableStateDB(ctrl)
	archiveFour := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		SkipPriming: true,
		Workers:     1,
	}

	// Simulate the execution of four requests in three blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			// Block 2
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 1, Data: reqBlockTwo})
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 2, Data: reqBlockTwo})
			// Block 3
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 3, Transaction: 1, Data: reqBlockThree})
			// Block 4
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 4, Transaction: 0, Data: reqBlockFour})
			return nil
		})

	// The expectation is that all of those requests are properly executed.
	// Since we are running sequential mode with 1 worker, they all need to be in order.
	gomock.InOrder(
		// Req 1
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveOne, nil),
		archiveOne.EXPECT().BeginTransaction(uint32(1)),
		archiveOne.EXPECT().GetBalance(common.HexToAddress(testingAddress)).Return(new(big.Int).SetInt64(1)),
		archiveOne.EXPECT().EndTransaction(),
		archiveOne.EXPECT().Release(),
		// Req 2
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveTwo, nil),
		archiveTwo.EXPECT().BeginTransaction(uint32(2)),
		archiveTwo.EXPECT().GetBalance(common.HexToAddress(testingAddress)).Return(new(big.Int).SetInt64(1)),
		archiveTwo.EXPECT().EndTransaction(),
		archiveTwo.EXPECT().Release(),
		// Req 3
		db.EXPECT().GetArchiveState(uint64(3)).Return(archiveThree, nil),
		archiveThree.EXPECT().BeginTransaction(uint32(1)),
		archiveThree.EXPECT().GetNonce(common.HexToAddress(testingAddress)).Return(uint64(1)),
		archiveThree.EXPECT().EndTransaction(),
		archiveThree.EXPECT().Release(),
		// Req 4
		db.EXPECT().GetArchiveState(uint64(4)).Return(archiveFour, nil),
		archiveFour.EXPECT().BeginTransaction(uint32(0)),
		archiveFour.EXPECT().GetCode(common.HexToAddress(testingAddress)).Return(hexutil.MustDecode("0x10")),
		archiveFour.EXPECT().EndTransaction(),
		archiveFour.EXPECT().Release(),
	)

	if err := run(cfg, provider, db, rpcProcessor{cfg}, nil); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestRpc_AllDbEventsAreIssuedInOrder_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveOne := state.NewMockNonCommittableStateDB(ctrl)
	archiveTwo := state.NewMockNonCommittableStateDB(ctrl)
	archiveThree := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		SkipPriming: true,
		Workers:     4,
	}

	// Simulate the execution of four requests in three blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			// Block 2
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 1, Data: reqBlockTwo})
			// Block 3
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 3, Transaction: 2, Data: reqBlockThree})
			// Block 4
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 4, Transaction: 0, Data: reqBlockFour})
			return nil
		})

	// The expectation is that all of those requests are properly executed.
	// Since we are running sequential mode with 1 worker, they all need to be in order.
	gomock.InOrder(
		// Req 1
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveOne, nil),
		archiveOne.EXPECT().BeginTransaction(uint32(1)),
		archiveOne.EXPECT().GetBalance(common.HexToAddress(testingAddress)).Return(new(big.Int).SetInt64(1)),
		archiveOne.EXPECT().EndTransaction(),
		archiveOne.EXPECT().Release(),
	)
	gomock.InOrder(
		// Req 2
		db.EXPECT().GetArchiveState(uint64(3)).Return(archiveTwo, nil),
		archiveTwo.EXPECT().BeginTransaction(uint32(2)),
		archiveTwo.EXPECT().GetNonce(common.HexToAddress(testingAddress)).Return(uint64(3)),
		archiveTwo.EXPECT().EndTransaction(),
		archiveTwo.EXPECT().Release(),
	)
	gomock.InOrder(
		// Req 3
		db.EXPECT().GetArchiveState(uint64(4)).Return(archiveThree, nil),
		archiveThree.EXPECT().BeginTransaction(uint32(0)),
		archiveThree.EXPECT().GetCode(common.HexToAddress(testingAddress)).Return(hexutil.MustDecode("0x10")),
		archiveThree.EXPECT().EndTransaction(),
		archiveThree.EXPECT().Release(),
	)

	if err := run(cfg, provider, db, rpcProcessor{cfg}, nil); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestRpc_AllTransactionsAreProcessedInOrder_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)
	ext := executor.NewMockExtension[*rpc.RequestAndResults](ctrl)
	processor := executor.NewMockProcessor[*rpc.RequestAndResults](ctrl)

	config := &utils.Config{
		First:    2,
		Last:     4,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
		Workers:  1,
	}

	// Simulate the execution of four requests in three blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			// Block 2
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 1, Data: reqBlockTwo})
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 2, Data: reqBlockTwo})
			// Block 3
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 3, Transaction: 1, Data: reqBlockThree})
			// Block 4
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 4, Transaction: 0, Data: reqBlockFour})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all blocks and transactions need to be in order.

	gomock.InOrder(
		ext.EXPECT().PreRun(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),

		// Req 1
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		archive.EXPECT().EndTransaction(),
		archive.EXPECT().Release(),

		// Req 2
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(2)),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		archive.EXPECT().EndTransaction(),
		archive.EXPECT().Release(),

		// Req 3
		db.EXPECT().GetArchiveState(uint64(3)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](3), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](3), gomock.Any()),
		archive.EXPECT().EndTransaction(),
		archive.EXPECT().Release(),

		// Block 4
		db.EXPECT().GetArchiveState(uint64(4)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(0)),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](4), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](4), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](4), gomock.Any()),
		archive.EXPECT().EndTransaction(),
		archive.EXPECT().Release(),

		ext.EXPECT().PostRun(executor.AtBlock[*rpc.RequestAndResults](5), gomock.Any(), nil),
	)

	if err := run(config, provider, db, processor, []executor.Extension[*rpc.RequestAndResults]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestRpc_AllTransactionsAreProcessed_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	db := state.NewMockStateDB(ctrl)
	archiveOne := state.NewMockNonCommittableStateDB(ctrl)
	archiveTwo := state.NewMockNonCommittableStateDB(ctrl)
	archiveThree := state.NewMockNonCommittableStateDB(ctrl)
	ext := executor.NewMockExtension[*rpc.RequestAndResults](ctrl)
	processor := executor.NewMockProcessor[*rpc.RequestAndResults](ctrl)

	config := &utils.Config{
		First:    2,
		Last:     4,
		ChainID:  utils.MainnetChainID,
		LogLevel: "Critical",
		Workers:  2,
	}

	// Simulate the execution of four requests in three blocks.
	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			// Block 2
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 1, Data: reqBlockTwo})
			// Block 3
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 3, Transaction: 2, Data: reqBlockThree})
			// Block 4
			consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 4, Transaction: 0, Data: reqBlockFour})
			return nil
		})

	// The expectation is that all of those blocks and transactions
	// are properly opened, prepared, executed, and closed.
	// Since we are running sequential mode with 1 worker,
	// all blocks and transactions need to be in order.

	pre := ext.EXPECT().PreRun(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any())
	post := ext.EXPECT().PostRun(executor.AtBlock[*rpc.RequestAndResults](5), gomock.Any(), nil)

	gomock.InOrder(
		pre,
		// Req 1
		db.EXPECT().GetArchiveState(uint64(2)).Return(archiveOne, nil),
		archiveOne.EXPECT().BeginTransaction(uint32(1)),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](2), gomock.Any()),
		archiveOne.EXPECT().EndTransaction(),
		archiveOne.EXPECT().Release(),
		post,
	)

	gomock.InOrder(
		pre,
		// Req 2
		db.EXPECT().GetArchiveState(uint64(3)).Return(archiveTwo, nil),
		archiveTwo.EXPECT().BeginTransaction(uint32(2)),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](3), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](3), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](3), gomock.Any()),
		archiveTwo.EXPECT().EndTransaction(),
		archiveTwo.EXPECT().Release(),
		post,
	)

	gomock.InOrder(
		pre,
		// Req 3
		db.EXPECT().GetArchiveState(uint64(4)).Return(archiveThree, nil),
		archiveThree.EXPECT().BeginTransaction(uint32(0)),
		ext.EXPECT().PreTransaction(executor.AtBlock[*rpc.RequestAndResults](4), gomock.Any()),
		processor.EXPECT().Process(executor.AtBlock[*rpc.RequestAndResults](4), gomock.Any()),
		ext.EXPECT().PostTransaction(executor.AtBlock[*rpc.RequestAndResults](4), gomock.Any()),
		archiveThree.EXPECT().EndTransaction(),
		archiveThree.EXPECT().Release(),
		post,
	)

	if err := run(config, provider, db, processor, []executor.Extension[*rpc.RequestAndResults]{ext}); err != nil {
		t.Errorf("run failed: %v", err)
	}
}

func TestRpc_ValidationDoesNotFailOnValidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		Validate:    true,
		SkipPriming: true,
		Workers:     1,
	}

	var err error
	reqBlockTwo.Response.Result, err = json.Marshal("0x1")
	if err != nil {
		t.Fatalf("cannot marshal result; %v", err)
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			return consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 1, Data: reqBlockTwo})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().GetBalance(common.HexToAddress(testingAddress)).Return(new(big.Int).SetInt64(1)),
		archive.EXPECT().EndTransaction(),
		archive.EXPECT().Release(),
	)

	// run fails but not on validation
	err = run(cfg, provider, db, rpcProcessor{cfg}, nil)
	if err != nil {
		t.Errorf("run must not fail")
	}
}

func TestRpc_ValidationDoesNotFailOnValidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		Validate:    true,
		SkipPriming: true,
		Workers:     4,
	}

	var err error
	reqBlockTwo.Response.Result, err = json.Marshal("0x1")
	if err != nil {
		t.Fatalf("cannot marshal result; %v", err)
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			return consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 1, Data: reqBlockTwo})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().GetBalance(common.HexToAddress(testingAddress)).Return(new(big.Int).SetInt64(1)),
		archive.EXPECT().EndTransaction(),
		archive.EXPECT().Release(),
	)

	// run fails but not on validation
	err = run(cfg, provider, db, rpcProcessor{cfg}, nil)
	if err != nil {
		t.Errorf("run must not fail")
	}
}

func TestRpc_ValidationFailsOnValidTransaction_Sequential(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		Validate:    true,
		SkipPriming: true,
		Workers:     1,
	}

	var err error
	reqBlockTwo.Response.Result, err = json.Marshal("0x1")
	if err != nil {
		t.Fatalf("cannot marshal result; %v", err)
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			return consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 1, Data: reqBlockTwo})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().GetBalance(common.HexToAddress(testingAddress)).Return(new(big.Int).SetInt64(2)),
		archive.EXPECT().EndTransaction(),
		archive.EXPECT().Release(),
	)

	// run fails but not on validation
	err = run(cfg, provider, db, rpcProcessor{cfg}, nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	if !strings.Contains(err.Error(), "result do not match") {
		t.Fatalf("unexpected err %v", err)
	}
}

func TestRpc_ValidationFailsOnValidTransaction_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := executor.NewMockProvider[*rpc.RequestAndResults](ctrl)
	db := state.NewMockStateDB(ctrl)
	archive := state.NewMockNonCommittableStateDB(ctrl)

	cfg := &utils.Config{
		First:       2,
		Last:        4,
		ChainID:     utils.MainnetChainID,
		Validate:    true,
		SkipPriming: true,
		Workers:     4,
	}

	var err error
	reqBlockTwo.Response.Result, err = json.Marshal("0x1")
	if err != nil {
		t.Fatalf("cannot marshal result; %v", err)
	}

	provider.EXPECT().
		Run(2, 5, gomock.Any()).
		DoAndReturn(func(_ int, _ int, consumer executor.Consumer[*rpc.RequestAndResults]) error {
			return consumer(executor.TransactionInfo[*rpc.RequestAndResults]{Block: 2, Transaction: 1, Data: reqBlockTwo})
		})

	gomock.InOrder(
		db.EXPECT().GetArchiveState(uint64(2)).Return(archive, nil),
		archive.EXPECT().BeginTransaction(uint32(1)),
		archive.EXPECT().GetBalance(common.HexToAddress(testingAddress)).Return(new(big.Int).SetInt64(2)),
		archive.EXPECT().EndTransaction(),
		archive.EXPECT().Release(),
	)

	// run fails but not on validation
	err = run(cfg, provider, db, rpcProcessor{cfg}, nil)
	if err == nil {
		t.Errorf("run must fail")
	}

	if !strings.Contains(err.Error(), "result do not match") {
		t.Fatalf("unexpected err %v", err)
	}
}

var reqBlockTwo = &rpc.RequestAndResults{
	RequestedBlock: 2,
	Query: &rpc.Body{
		Version:    "2.0",
		ID:         json.RawMessage{1},
		Params:     []interface{}{testingAddress, "0x2"},
		Method:     "eth_getBalance",
		Namespace:  "eth",
		MethodBase: "getBalance",
	},
	Response: &rpc.Response{
		Version:   "2.0",
		ID:        json.RawMessage{1},
		BlockID:   10,
		Timestamp: 10,
	},
}

var reqBlockThree = &rpc.RequestAndResults{
	RequestedBlock: 3,
	Query: &rpc.Body{
		Version:    "2.0",
		ID:         json.RawMessage{1},
		Params:     []interface{}{testingAddress, "0x3"},
		Method:     "eth_getTransactionCount",
		Namespace:  "eth",
		MethodBase: "getTransactionCount",
	},
	Response: &rpc.Response{
		Version:   "2.0",
		ID:        json.RawMessage{1},
		BlockID:   10,
		Timestamp: 10,
	},
}

var reqBlockFour = &rpc.RequestAndResults{
	RequestedBlock: 4,
	Query: &rpc.Body{
		Version:    "2.0",
		ID:         json.RawMessage{1},
		Params:     []interface{}{testingAddress, "0x4"},
		Method:     "eth_getCode",
		Namespace:  "eth",
		MethodBase: "getCode",
	},
	Response: &rpc.Response{
		Version:   "2.0",
		ID:        json.RawMessage{1},
		BlockID:   10,
		Timestamp: 10,
	},
}
