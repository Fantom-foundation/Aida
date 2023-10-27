// Code generated by MockGen. DO NOT EDIT.
// Source: reader.go
//
// Generated by this command:
//
//	mockgen -source reader.go -destination reader_mocks.go -package rpc_iterator
//
// Package rpc_iterator is a generated GoMock package.
package rpc_iterator

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockRPCIterator is a mock of RPCIterator interface.
type MockRPCIterator struct {
	ctrl     *gomock.Controller
	recorder *MockRPCIteratorMockRecorder
}

// MockRPCIteratorMockRecorder is the mock recorder for MockRPCIterator.
type MockRPCIteratorMockRecorder struct {
	mock *MockRPCIterator
}

// NewMockRPCIterator creates a new mock instance.
func NewMockRPCIterator(ctrl *gomock.Controller) *MockRPCIterator {
	mock := &MockRPCIterator{ctrl: ctrl}
	mock.recorder = &MockRPCIteratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRPCIterator) EXPECT() *MockRPCIteratorMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockRPCIterator) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockRPCIteratorMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockRPCIterator)(nil).Close))
}

// Error mocks base method.
func (m *MockRPCIterator) Error() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Error")
	ret0, _ := ret[0].(error)
	return ret0
}

// Error indicates an expected call of Error.
func (mr *MockRPCIteratorMockRecorder) Error() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockRPCIterator)(nil).Error))
}

// Next mocks base method.
func (m *MockRPCIterator) Next() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Next indicates an expected call of Next.
func (mr *MockRPCIteratorMockRecorder) Next() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockRPCIterator)(nil).Next))
}

// Value mocks base method.
func (m *MockRPCIterator) Value() *RequestWithResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Value")
	ret0, _ := ret[0].(*RequestWithResponse)
	return ret0
}

// Value indicates an expected call of Value.
func (mr *MockRPCIteratorMockRecorder) Value() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Value", reflect.TypeOf((*MockRPCIterator)(nil).Value))
}
