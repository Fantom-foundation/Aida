// Code generated by MockGen. DO NOT EDIT.
// Source: vm_statistics_printer_test.go

// Package extension is a generated GoMock package.
package profiler

import (
	reflect "reflect"

	vm "github.com/ethereum/go-ethereum/core/vm"
	gomock "go.uber.org/mock/gomock"
)

// MockProfilingVm is a mock of ProfilingVm interface.
type MockProfilingVm struct {
	ctrl     *gomock.Controller
	recorder *MockProfilingVmMockRecorder
}

// MockProfilingVmMockRecorder is the mock recorder for MockProfilingVm.
type MockProfilingVmMockRecorder struct {
	mock *MockProfilingVm
}

// NewMockProfilingVm creates a new mock instance.
func NewMockProfilingVm(ctrl *gomock.Controller) *MockProfilingVm {
	mock := &MockProfilingVm{ctrl: ctrl}
	mock.recorder = &MockProfilingVmMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProfilingVm) EXPECT() *MockProfilingVmMockRecorder {
	return m.recorder
}

// DumpProfile mocks base method.
func (m *MockProfilingVm) DumpProfile() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DumpProfile")
}

// DumpProfile indicates an expected call of DumpProfile.
func (mr *MockProfilingVmMockRecorder) DumpProfile() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DumpProfile", reflect.TypeOf((*MockProfilingVm)(nil).DumpProfile))
}

// NewInterpreter mocks base method.
func (m *MockProfilingVm) NewInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewInterpreter", evm, cfg)
	ret0, _ := ret[0].(vm.EVMInterpreter)
	return ret0
}

// NewInterpreter indicates an expected call of NewInterpreter.
func (mr *MockProfilingVmMockRecorder) NewInterpreter(evm, cfg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewInterpreter", reflect.TypeOf((*MockProfilingVm)(nil).NewInterpreter), evm, cfg)
}

// ResetProfile mocks base method.
func (m *MockProfilingVm) ResetProfile() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ResetProfile")
}

// ResetProfile indicates an expected call of ResetProfile.
func (mr *MockProfilingVmMockRecorder) ResetProfile() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetProfile", reflect.TypeOf((*MockProfilingVm)(nil).ResetProfile))
}