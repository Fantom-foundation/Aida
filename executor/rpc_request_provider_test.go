package executor

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Aida/rpc"
	"github.com/Fantom-foundation/Aida/utils"
	"go.uber.org/mock/gomock"
)

func TestRPCRequestProvider_WorksWithValidResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockRPCReqConsumer(ctrl)
	i := rpc.NewMockIterator(ctrl)

	cfg := &utils.Config{}

	provider := openRpcRecording(i, cfg, nil)

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

	provider := openRpcRecording(i, cfg, nil)

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

	provider := openRpcRecording(i, cfg, nil)

	defer provider.Close()

	gomock.InOrder(
		i.EXPECT().Next().Return(true),
		i.EXPECT().Error().Return(nil),
		i.EXPECT().Value().Return(nil),
		i.EXPECT().Close(),
	)

	if err := provider.Run(10, 11, toRPCConsumer(consumer)); err != nil {
		t.Fatalf("failed to iterate through requests: %v", err)
	}
}

func TestRPCRequestProvider_ErrorReturnedByIteratorEndsTheApp(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockRPCReqConsumer(ctrl)
	i := rpc.NewMockIterator(ctrl)

	cfg := &utils.Config{}

	provider := openRpcRecording(i, cfg, nil)

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

var validResp = &rpc.RequestAndResults{
	Block:       10,
	Timestamp:   10,
	Query:       nil,
	Response:    json.RawMessage{},
	Error:       nil,
	ParamsRaw:   nil,
	ResponseRaw: nil,
}

var errResp = &rpc.RequestAndResults{
	Block:     10,
	Timestamp: 10,
	Query:     nil,
	Response:  nil,
	Error: &rpc.Error{
		Code:    -1,
		Message: "err",
	},
	ParamsRaw:   nil,
	ResponseRaw: nil,
}