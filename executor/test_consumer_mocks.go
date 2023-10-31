// Code generated by MockGen. DO NOT EDIT.
// Source: test_consumer.go
//
// Generated by this command:
//
//	mockgen -source test_consumer.go -destination test_consumer_mocks.go -package executor
//
// Package executor is a generated GoMock package.
package executor

import (
	reflect "reflect"

	rpc_iterator "github.com/Fantom-foundation/Aida/rpc"
	substate "github.com/Fantom-foundation/Substate"
	gomock "go.uber.org/mock/gomock"
)

// MockTxConsumer is a mock of TxConsumer interface.
type MockTxConsumer struct {
	ctrl     *gomock.Controller
	recorder *MockTxConsumerMockRecorder
}

// MockTxConsumerMockRecorder is the mock recorder for MockTxConsumer.
type MockTxConsumerMockRecorder struct {
	mock *MockTxConsumer
}

// NewMockTxConsumer creates a new mock instance.
func NewMockTxConsumer(ctrl *gomock.Controller) *MockTxConsumer {
	mock := &MockTxConsumer{ctrl: ctrl}
	mock.recorder = &MockTxConsumerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTxConsumer) EXPECT() *MockTxConsumerMockRecorder {
	return m.recorder
}

// Consume mocks base method.
func (m *MockTxConsumer) Consume(block, transaction int, substate *substate.Substate) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Consume", block, transaction, substate)
	ret0, _ := ret[0].(error)
	return ret0
}

// Consume indicates an expected call of Consume.
func (mr *MockTxConsumerMockRecorder) Consume(block, transaction, substate any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Consume", reflect.TypeOf((*MockTxConsumer)(nil).Consume), block, transaction, substate)
}

// MockRPCReqConsumer is a mock of RPCReqConsumer interface.
type MockRPCReqConsumer struct {
	ctrl     *gomock.Controller
	recorder *MockRPCReqConsumerMockRecorder
}

// MockRPCReqConsumerMockRecorder is the mock recorder for MockRPCReqConsumer.
type MockRPCReqConsumerMockRecorder struct {
	mock *MockRPCReqConsumer
}

// NewMockRPCReqConsumer creates a new mock instance.
func NewMockRPCReqConsumer(ctrl *gomock.Controller) *MockRPCReqConsumer {
	mock := &MockRPCReqConsumer{ctrl: ctrl}
	mock.recorder = &MockRPCReqConsumerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRPCReqConsumer) EXPECT() *MockRPCReqConsumerMockRecorder {
	return m.recorder
}

// Consume mocks base method.
func (m *MockRPCReqConsumer) Consume(block, transaction int, req *rpc_iterator.RequestAndResults) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Consume", block, transaction, req)
	ret0, _ := ret[0].(error)
	return ret0
}

// Consume indicates an expected call of Consume.
func (mr *MockRPCReqConsumerMockRecorder) Consume(block, transaction, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Consume", reflect.TypeOf((*MockRPCReqConsumer)(nil).Consume), block, transaction, req)
}
