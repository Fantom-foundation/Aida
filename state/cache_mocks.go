// Code generated by MockGen. DO NOT EDIT.
// Source: cache.go
//
// Generated by this command:
//
//	mockgen -source cache.go -destination cache_mocks.go -package state
//

// Package state is a generated GoMock package.
package state

import (
	reflect "reflect"

	common "github.com/ethereum/go-ethereum/common"
	gomock "go.uber.org/mock/gomock"
)

// MockCodeCache is a mock of CodeCache interface.
type MockCodeCache struct {
	ctrl     *gomock.Controller
	recorder *MockCodeCacheMockRecorder
}

// MockCodeCacheMockRecorder is the mock recorder for MockCodeCache.
type MockCodeCacheMockRecorder struct {
	mock *MockCodeCache
}

// NewMockCodeCache creates a new mock instance.
func NewMockCodeCache(ctrl *gomock.Controller) *MockCodeCache {
	mock := &MockCodeCache{ctrl: ctrl}
	mock.recorder = &MockCodeCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCodeCache) EXPECT() *MockCodeCacheMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockCodeCache) Get(addr common.Address, code []byte) common.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", addr, code)
	ret0, _ := ret[0].(common.Hash)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockCodeCacheMockRecorder) Get(addr, code any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCodeCache)(nil).Get), addr, code)
}
