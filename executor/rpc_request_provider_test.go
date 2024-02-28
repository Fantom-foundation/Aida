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

	provider := openRpcRecording(i, cfg, nil, nil, []string{"testfile"})

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

	provider := openRpcRecording(i, cfg, nil, nil, []string{"testfile"})

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

	provider := openRpcRecording(i, cfg, nil, nil, []string{"testfile"})

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

	provider := openRpcRecording(i, cfg, nil, nil, []string{"testfile"})

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

	provider := openRpcRecording(i, cfg, nil, nil, []string{"testfile"})

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

func TestRPCRequestProvider_ReportsLastFileWhenError(t *testing.T) {
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
