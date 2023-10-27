// Code generated by MockGen. DO NOT EDIT.
// Source: executor.go
//
// Generated by this command:
//
//	mockgen -source executor.go -destination executor_mocks.go -package executor
//
// Package executor is a generated GoMock package.
package executor

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockExecutor is a mock of Executor interface.
type MockExecutor[T any] struct {
	ctrl     *gomock.Controller
	recorder *MockExecutorMockRecorder[T]
}

// MockExecutorMockRecorder is the mock recorder for MockExecutor.
type MockExecutorMockRecorder[T any] struct {
	mock *MockExecutor[T]
}

// NewMockExecutor creates a new mock instance.
func NewMockExecutor[T any](ctrl *gomock.Controller) *MockExecutor[T] {
	mock := &MockExecutor[T]{ctrl: ctrl}
	mock.recorder = &MockExecutorMockRecorder[T]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutor[T]) EXPECT() *MockExecutorMockRecorder[T] {
	return m.recorder
}

// Run mocks base method.
func (m *MockExecutor[T]) Run(params Params, processor Processor[T], extensions []Extension[T]) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", params, processor, extensions)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockExecutorMockRecorder[T]) Run(params, processor, extensions any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockExecutor[T])(nil).Run), params, processor, extensions)
}

// MockProcessor is a mock of Processor interface.
type MockProcessor[T any] struct {
	ctrl     *gomock.Controller
	recorder *MockProcessorMockRecorder[T]
}

// MockProcessorMockRecorder is the mock recorder for MockProcessor.
type MockProcessorMockRecorder[T any] struct {
	mock *MockProcessor[T]
}

// NewMockProcessor creates a new mock instance.
func NewMockProcessor[T any](ctrl *gomock.Controller) *MockProcessor[T] {
	mock := &MockProcessor[T]{ctrl: ctrl}
	mock.recorder = &MockProcessorMockRecorder[T]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProcessor[T]) EXPECT() *MockProcessorMockRecorder[T] {
	return m.recorder
}

// Process mocks base method.
func (m *MockProcessor[T]) Process(arg0 State[T], arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Process", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Process indicates an expected call of Process.
func (mr *MockProcessorMockRecorder[T]) Process(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Process", reflect.TypeOf((*MockProcessor[T])(nil).Process), arg0, arg1)
}

// MockExtension is a mock of Extension interface.
type MockExtension[T any] struct {
	ctrl     *gomock.Controller
	recorder *MockExtensionMockRecorder[T]
}

// MockExtensionMockRecorder is the mock recorder for MockExtension.
type MockExtensionMockRecorder[T any] struct {
	mock *MockExtension[T]
}

// NewMockExtension creates a new mock instance.
func NewMockExtension[T any](ctrl *gomock.Controller) *MockExtension[T] {
	mock := &MockExtension[T]{ctrl: ctrl}
	mock.recorder = &MockExtensionMockRecorder[T]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExtension[T]) EXPECT() *MockExtensionMockRecorder[T] {
	return m.recorder
}

// PostBlock mocks base method.
func (m *MockExtension[T]) PostBlock(arg0 State[T], arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostBlock", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostBlock indicates an expected call of PostBlock.
func (mr *MockExtensionMockRecorder[T]) PostBlock(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostBlock", reflect.TypeOf((*MockExtension[T])(nil).PostBlock), arg0, arg1)
}

// PostRun mocks base method.
func (m *MockExtension[T]) PostRun(arg0 State[T], arg1 *Context, arg2 error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostRun", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostRun indicates an expected call of PostRun.
func (mr *MockExtensionMockRecorder[T]) PostRun(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostRun", reflect.TypeOf((*MockExtension[T])(nil).PostRun), arg0, arg1, arg2)
}

// PostTransaction mocks base method.
func (m *MockExtension[T]) PostTransaction(arg0 State[T], arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostTransaction", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostTransaction indicates an expected call of PostTransaction.
func (mr *MockExtensionMockRecorder[T]) PostTransaction(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostTransaction", reflect.TypeOf((*MockExtension[T])(nil).PostTransaction), arg0, arg1)
}

// PreBlock mocks base method.
func (m *MockExtension[T]) PreBlock(arg0 State[T], arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreBlock", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreBlock indicates an expected call of PreBlock.
func (mr *MockExtensionMockRecorder[T]) PreBlock(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreBlock", reflect.TypeOf((*MockExtension[T])(nil).PreBlock), arg0, arg1)
}

// PreRun mocks base method.
func (m *MockExtension[T]) PreRun(arg0 State[T], arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreRun", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreRun indicates an expected call of PreRun.
func (mr *MockExtensionMockRecorder[T]) PreRun(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreRun", reflect.TypeOf((*MockExtension[T])(nil).PreRun), arg0, arg1)
}

// PreTransaction mocks base method.
func (m *MockExtension[T]) PreTransaction(arg0 State[T], arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreTransaction", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreTransaction indicates an expected call of PreTransaction.
func (mr *MockExtensionMockRecorder[T]) PreTransaction(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreTransaction", reflect.TypeOf((*MockExtension[T])(nil).PreTransaction), arg0, arg1)
}
