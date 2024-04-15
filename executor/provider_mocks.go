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

// Code generated by MockGen. DO NOT EDIT.
// Source: provider.go
//
// Generated by this command:
//
//	mockgen -source provider.go -destination provider_mocks.go -package executor
//
// Package executor is a generated GoMock package.
package executor

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockProvider is a mock of Provider interface.
type MockProvider[T any] struct {
	ctrl     *gomock.Controller
	recorder *MockProviderMockRecorder[T]
}

// MockProviderMockRecorder is the mock recorder for MockProvider.
type MockProviderMockRecorder[T any] struct {
	mock *MockProvider[T]
}

// NewMockProvider creates a new mock instance.
func NewMockProvider[T any](ctrl *gomock.Controller) *MockProvider[T] {
	mock := &MockProvider[T]{ctrl: ctrl}
	mock.recorder = &MockProviderMockRecorder[T]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProvider[T]) EXPECT() *MockProviderMockRecorder[T] {
	return m.recorder
}

// Close mocks base method.
func (m *MockProvider[T]) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockProviderMockRecorder[T]) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockProvider[T])(nil).Close))
}

// Run mocks base method.
func (m *MockProvider[T]) Run(from, to int, consumer Consumer[T]) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", from, to, consumer)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockProviderMockRecorder[T]) Run(from, to, consumer any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockProvider[T])(nil).Run), from, to, consumer)
}
