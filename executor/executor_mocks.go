// Code generated by MockGen. DO NOT EDIT.
// Source: executor.go

// Package executor is a generated GoMock package.
package executor

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockExecutor is a mock of Executor interface.
type MockExecutor struct {
	ctrl     *gomock.Controller
	recorder *MockExecutorMockRecorder
}

// MockExecutorMockRecorder is the mock recorder for MockExecutor.
type MockExecutorMockRecorder struct {
	mock *MockExecutor
}

// NewMockExecutor creates a new mock instance.
func NewMockExecutor(ctrl *gomock.Controller) *MockExecutor {
	mock := &MockExecutor{ctrl: ctrl}
	mock.recorder = &MockExecutorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutor) EXPECT() *MockExecutorMockRecorder {
	return m.recorder
}

// Run mocks base method.
func (m *MockExecutor) Run(params Params, processor Processor, extensions []Extension) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", params, processor, extensions)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockExecutorMockRecorder) Run(params, processor, extensions interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockExecutor)(nil).Run), params, processor, extensions)
}

// MockProcessor is a mock of Processor interface.
type MockProcessor struct {
	ctrl     *gomock.Controller
	recorder *MockProcessorMockRecorder
}

// MockProcessorMockRecorder is the mock recorder for MockProcessor.
type MockProcessorMockRecorder struct {
	mock *MockProcessor
}

// NewMockProcessor creates a new mock instance.
func NewMockProcessor(ctrl *gomock.Controller) *MockProcessor {
	mock := &MockProcessor{ctrl: ctrl}
	mock.recorder = &MockProcessorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProcessor) EXPECT() *MockProcessorMockRecorder {
	return m.recorder
}

// Process mocks base method.
func (m *MockProcessor) Process(arg0 State, arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Process", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Process indicates an expected call of Process.
func (mr *MockProcessorMockRecorder) Process(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Process", reflect.TypeOf((*MockProcessor)(nil).Process), arg0, arg1)
}

// MockExtension is a mock of Extension interface.
type MockExtension struct {
	ctrl     *gomock.Controller
	recorder *MockExtensionMockRecorder
}

// MockExtensionMockRecorder is the mock recorder for MockExtension.
type MockExtensionMockRecorder struct {
	mock *MockExtension
}

// NewMockExtension creates a new mock instance.
func NewMockExtension(ctrl *gomock.Controller) *MockExtension {
	mock := &MockExtension{ctrl: ctrl}
	mock.recorder = &MockExtensionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExtension) EXPECT() *MockExtensionMockRecorder {
	return m.recorder
}

// PostBlock mocks base method.
func (m *MockExtension) PostBlock(arg0 State, arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostBlock", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostBlock indicates an expected call of PostBlock.
func (mr *MockExtensionMockRecorder) PostBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostBlock", reflect.TypeOf((*MockExtension)(nil).PostBlock), arg0, arg1)
}

// PostRun mocks base method.
func (m *MockExtension) PostRun(arg0 State, arg1 *Context, arg2 error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostRun", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostRun indicates an expected call of PostRun.
func (mr *MockExtensionMockRecorder) PostRun(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostRun", reflect.TypeOf((*MockExtension)(nil).PostRun), arg0, arg1, arg2)
}

// PostTransaction mocks base method.
func (m *MockExtension) PostAction(arg0 State, arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostAction", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostTransaction indicates an expected call of PostTransaction.
func (mr *MockExtensionMockRecorder) PostTransaction(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostAction", reflect.TypeOf((*MockExtension)(nil).PostAction), arg0, arg1)
}

// PreBlock mocks base method.
func (m *MockExtension) PreBlock(arg0 State, arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreBlock", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreBlock indicates an expected call of PreBlock.
func (mr *MockExtensionMockRecorder) PreBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreBlock", reflect.TypeOf((*MockExtension)(nil).PreBlock), arg0, arg1)
}

// PreRun mocks base method.
func (m *MockExtension) PreRun(arg0 State, arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreRun", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreRun indicates an expected call of PreRun.
func (mr *MockExtensionMockRecorder) PreRun(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreRun", reflect.TypeOf((*MockExtension)(nil).PreRun), arg0, arg1)
}

// PreTransaction mocks base method.
func (m *MockExtension) PreAction(arg0 State, arg1 *Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreAction", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreTransaction indicates an expected call of PreTransaction.
func (mr *MockExtensionMockRecorder) PreTransaction(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreAction", reflect.TypeOf((*MockExtension)(nil).PreAction), arg0, arg1)
}
