// Code generated by MockGen. DO NOT EDIT.
// Source: substate_provider.go

// Package executor is a generated GoMock package.
package executor

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockSubstateProvider is a mock of ActionProvider interface.
type MockSubstateProvider struct {
	ctrl     *gomock.Controller
	recorder *MockSubstateProviderMockRecorder
}

// MockSubstateProviderMockRecorder is the mock recorder for MockSubstateProvider.
type MockSubstateProviderMockRecorder struct {
	mock *MockSubstateProvider
}

// NewMockSubstateProvider creates a new mock instance.
func NewMockSubstateProvider(ctrl *gomock.Controller) *MockSubstateProvider {
	mock := &MockSubstateProvider{ctrl: ctrl}
	mock.recorder = &MockSubstateProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSubstateProvider) EXPECT() *MockSubstateProviderMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockSubstateProvider) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockSubstateProviderMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockSubstateProvider)(nil).Close))
}

// Run mocks base method.
func (m *MockSubstateProvider) Run(from, to int, consumer Consumer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", from, to, consumer)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockSubstateProviderMockRecorder) Run(from, to, consumer interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockSubstateProvider)(nil).Run), from, to, consumer)
}
