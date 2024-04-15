// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestRPCRequestProvider_WorksWithValidResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockRPCReqConsumer(ctrl)
	i := rpc.NewMockIterator(ctrl)

	cfg := &utils.Config{}

	provider := openRpcRecording(i, cfg, logger.NewLogger("critical", "rpc-provider-test"), nil, []string{"testfile"})

	defer provider.Close()

	gomock.InOrder(
		i.EXPECT().Next().Return(true),
		i.EXPECT().Error().Return(nil),
		i.EXPECT().Value().Return(validResp),
		consumer.EXPECT().Consume(10, gomock.Any(), validResp),
		i.EXPECT().Next().Return(false),
		i.EXPECT().Close(),
	)

	if err := provider.Run(10, 11, toRPCConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through requests: %v", err)
	}
}

func TestRPCRequestProvider_WorksWithErrorResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockRPCReqConsumer(ctrl)
	i := rpc.NewMockIterator(ctrl)

	cfg := &utils.Config{}

	provider := openRpcRecording(i, cfg, logger.NewLogger("critical", "rpc-provider-test"), nil, []string{"testfile"})

	defer provider.Close()

	gomock.InOrder(
		i.EXPECT().Next().Return(true),
		i.EXPECT().Error().Return(nil),
		i.EXPECT().Value().Return(errResp),
		consumer.EXPECT().Consume(10, gomock.Any(), errResp),
		i.EXPECT().Next().Return(false),
		i.EXPECT().Close(),
	)

	if err := provider.Run(10, 11, toRPCConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through requests: %v", err)
	}
}

func TestRPCRequestProvider_NilRequestDoesNotGetToConsumer(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockRPCReqConsumer(ctrl)
	i := rpc.NewMockIterator(ctrl)

	cfg := &utils.Config{}

	provider := openRpcRecording(i, cfg, logger.NewLogger("critical", "rpc-provider-test"), nil, []string{"testfile"})

	defer provider.Close()

	gomock.InOrder(
		i.EXPECT().Next().Return(true),
		i.EXPECT().Error().Return(nil),
		i.EXPECT().Value().Return(nil),
		i.EXPECT().Close(),
	)

	err := provider.Run(10, 11, toRPCConsumer(consumer))
	if err == nil {
		t.Fatal("provider must return error")
	}

	got := err.Error()
	want := "iterator returned nil request"

	if strings.Compare(got, want) != 0 {
		t.Fatalf("unexpected error\ngot: %v\nwant:%v", got, want)
	}

}

func TestRPCRequestProvider_ErrorReturnedByIteratorEndsTheApp(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockRPCReqConsumer(ctrl)
	i := rpc.NewMockIterator(ctrl)

	cfg := &utils.Config{}

	provider := openRpcRecording(i, cfg, logger.NewLogger("critical", "rpc-provider-test"), nil, []string{"testfile"})

	defer provider.Close()

	gomock.InOrder(
		i.EXPECT().Next().Return(true),
		i.EXPECT().Error().Return(errors.New("err")),
		i.EXPECT().Error().Return(errors.New("err")),
		i.EXPECT().Close(),
	)

	if err := provider.Run(10, 11, toRPCConsumer(consumer)); err == nil {
		if strings.Compare(err.Error(), "iterator returned error; err") != 0 {
			t.Fatal("unexpected error returned by the iterator")
		}
		t.Fatal("the test should return an error")
	}
}

func TestRPCRequestProvider_GetLogMethodDoesNotEndIteration(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockRPCReqConsumer(ctrl)
	i := rpc.NewMockIterator(ctrl)

	cfg := &utils.Config{}

	provider := openRpcRecording(i, cfg, logger.NewLogger("critical", "rpc-provider-test"), nil, []string{"testfile"})

	defer provider.Close()

	gomock.InOrder(
		i.EXPECT().Next().Return(true),
		i.EXPECT().Error().Return(nil),
		i.EXPECT().Value().Return(logResp),
		i.EXPECT().Next().Return(true),
		i.EXPECT().Error().Return(nil),
		i.EXPECT().Value().Return(logResp),
		i.EXPECT().Next().Return(false),
		i.EXPECT().Close(),
	)

	if err := provider.Run(10, 11, toRPCConsumer(consumer)); err != nil {
		t.Fatal("test cannot fail")
	}
}

func TestRPCRequestProvider_ReportsAboutRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockRPCReqConsumer(ctrl)
	log := logger.NewMockLogger(ctrl)
	i := rpc.NewMockIterator(ctrl)

	cfg := &utils.Config{}
	cfg.RpcRecordingPath = "test_file"

	provider := openRpcRecording(i, cfg, log, nil, []string{cfg.RpcRecordingPath})

	defer provider.Close()

	gomock.InOrder(
		i.EXPECT().Next().Return(true),
		i.EXPECT().Error().Return(nil),
		i.EXPECT().Value().Return(validResp),
		log.EXPECT().Noticef("Iterating file %v/%v path: %v", 1, 1, "test_file"),
		log.EXPECT().Noticef("First block of recording: %v", 10),
		consumer.EXPECT().Consume(10, gomock.Any(), validResp).Return(errors.New("err")),
		log.EXPECT().Infof("Last iterated file: %v", "test_file"),

		//i.EXPECT().Next().Return(false),
		i.EXPECT().Close(),
	)

	if err := provider.Run(10, 11, toRPCConsumer(consumer)); err == nil {
		t.Fatal("run must fail")
	}
}

var validResp = &rpc.RequestAndResults{
	Query: &rpc.Body{},
	Response: &rpc.Response{
		Version:   "2.0",
		ID:        json.RawMessage{1},
		BlockID:   10,
		Timestamp: 10,
		Result:    nil,
		Payload:   nil,
	},
	Error:       nil,
	ParamsRaw:   nil,
	ResponseRaw: nil,
}

var errResp = &rpc.RequestAndResults{
	Query:    &rpc.Body{},
	Response: nil,
	Error: &rpc.ErrorResponse{
		Version:   "2.0",
		Id:        json.RawMessage{1},
		BlockID:   10,
		Timestamp: 10,
		Error: rpc.ErrorMessage{
			Code:    -1,
			Message: "err",
		},
		Payload: nil,
	},
	ParamsRaw:   nil,
	ResponseRaw: nil,
}

var logResp = &rpc.RequestAndResults{
	Query: &rpc.Body{
		MethodBase: "getLogs",
	},
	Response: &rpc.Response{
		Version:   "2.0",
		ID:        json.RawMessage{1},
		BlockID:   10,
		Timestamp: 10,
		Result:    nil,
		Payload:   nil,
	},
	Error:       nil,
	ParamsRaw:   nil,
	ResponseRaw: nil,
}
