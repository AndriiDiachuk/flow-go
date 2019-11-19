// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/dapperlabs/flow-go/sdk/emulator (interfaces: EmulatedBlockchainAPI)

// Package mocks is a generated GoMock package.
package mocks

import (
	crypto "github.com/dapperlabs/flow-go/crypto"
	flow "github.com/dapperlabs/flow-go/model/flow"
	types "github.com/dapperlabs/flow-go/sdk/emulator/types"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockEmulatedBlockchainAPI is a mock of EmulatedBlockchainAPI interface
type MockEmulatedBlockchainAPI struct {
	ctrl     *gomock.Controller
	recorder *MockEmulatedBlockchainAPIMockRecorder
}

// MockEmulatedBlockchainAPIMockRecorder is the mock recorder for MockEmulatedBlockchainAPI
type MockEmulatedBlockchainAPIMockRecorder struct {
	mock *MockEmulatedBlockchainAPI
}

// NewMockEmulatedBlockchainAPI creates a new mock instance
func NewMockEmulatedBlockchainAPI(ctrl *gomock.Controller) *MockEmulatedBlockchainAPI {
	mock := &MockEmulatedBlockchainAPI{ctrl: ctrl}
	mock.recorder = &MockEmulatedBlockchainAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockEmulatedBlockchainAPI) EXPECT() *MockEmulatedBlockchainAPIMockRecorder {
	return m.recorder
}

// CommitBlock mocks base method
func (m *MockEmulatedBlockchainAPI) CommitBlock() (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CommitBlock")
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CommitBlock indicates an expected call of CommitBlock
func (mr *MockEmulatedBlockchainAPIMockRecorder) CommitBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitBlock", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).CommitBlock))
}

// CreateAccount mocks base method
func (m *MockEmulatedBlockchainAPI) CreateAccount(arg0 []flow.AccountPublicKey, arg1 []byte, arg2 uint64) (flow.Address, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAccount", arg0, arg1, arg2)
	ret0, _ := ret[0].(flow.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAccount indicates an expected call of CreateAccount
func (mr *MockEmulatedBlockchainAPIMockRecorder) CreateAccount(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAccount", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).CreateAccount), arg0, arg1, arg2)
}

// ExecuteScript mocks base method
func (m *MockEmulatedBlockchainAPI) ExecuteScript(arg0 []byte) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecuteScript", arg0)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecuteScript indicates an expected call of ExecuteScript
func (mr *MockEmulatedBlockchainAPIMockRecorder) ExecuteScript(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecuteScript", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).ExecuteScript), arg0)
}

// ExecuteScriptAtBlock mocks base method
func (m *MockEmulatedBlockchainAPI) ExecuteScriptAtBlock(arg0 []byte, arg1 uint64) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecuteScriptAtBlock", arg0, arg1)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecuteScriptAtBlock indicates an expected call of ExecuteScriptAtBlock
func (mr *MockEmulatedBlockchainAPIMockRecorder) ExecuteScriptAtBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecuteScriptAtBlock", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).ExecuteScriptAtBlock), arg0, arg1)
}

// GetAccount mocks base method
func (m *MockEmulatedBlockchainAPI) GetAccount(arg0 flow.Address) (*flow.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccount", arg0)
	ret0, _ := ret[0].(*flow.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccount indicates an expected call of GetAccount
func (mr *MockEmulatedBlockchainAPIMockRecorder) GetAccount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccount", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).GetAccount), arg0)
}

// GetAccountAtBlock mocks base method
func (m *MockEmulatedBlockchainAPI) GetAccountAtBlock(arg0 flow.Address, arg1 uint64) (*flow.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccountAtBlock", arg0, arg1)
	ret0, _ := ret[0].(*flow.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccountAtBlock indicates an expected call of GetAccountAtBlock
func (mr *MockEmulatedBlockchainAPIMockRecorder) GetAccountAtBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccountAtBlock", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).GetAccountAtBlock), arg0, arg1)
}

// GetBlockByHash mocks base method
func (m *MockEmulatedBlockchainAPI) GetBlockByHash(arg0 crypto.Hash) (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockByHash", arg0)
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockByHash indicates an expected call of GetBlockByHash
func (mr *MockEmulatedBlockchainAPIMockRecorder) GetBlockByHash(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByHash", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).GetBlockByHash), arg0)
}

// GetBlockByNumber mocks base method
func (m *MockEmulatedBlockchainAPI) GetBlockByNumber(arg0 uint64) (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockByNumber", arg0)
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockByNumber indicates an expected call of GetBlockByNumber
func (mr *MockEmulatedBlockchainAPIMockRecorder) GetBlockByNumber(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByNumber", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).GetBlockByNumber), arg0)
}

// GetLatestBlock mocks base method
func (m *MockEmulatedBlockchainAPI) GetLatestBlock() (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestBlock")
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestBlock indicates an expected call of GetLatestBlock
func (mr *MockEmulatedBlockchainAPIMockRecorder) GetLatestBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestBlock", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).GetLatestBlock))
}

// GetTransaction mocks base method
func (m *MockEmulatedBlockchainAPI) GetTransaction(arg0 crypto.Hash) (*flow.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransaction", arg0)
	ret0, _ := ret[0].(*flow.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTransaction indicates an expected call of GetTransaction
func (mr *MockEmulatedBlockchainAPIMockRecorder) GetTransaction(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransaction", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).GetTransaction), arg0)
}

// LastCreatedAccount mocks base method
func (m *MockEmulatedBlockchainAPI) LastCreatedAccount() flow.Account {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LastCreatedAccount")
	ret0, _ := ret[0].(flow.Account)
	return ret0
}

// LastCreatedAccount indicates an expected call of LastCreatedAccount
func (mr *MockEmulatedBlockchainAPIMockRecorder) LastCreatedAccount() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastCreatedAccount", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).LastCreatedAccount))
}

// RootAccountAddress mocks base method
func (m *MockEmulatedBlockchainAPI) RootAccountAddress() flow.Address {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RootAccountAddress")
	ret0, _ := ret[0].(flow.Address)
	return ret0
}

// RootAccountAddress indicates an expected call of RootAccountAddress
func (mr *MockEmulatedBlockchainAPIMockRecorder) RootAccountAddress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RootAccountAddress", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).RootAccountAddress))
}

// RootKey mocks base method
func (m *MockEmulatedBlockchainAPI) RootKey() flow.AccountPrivateKey {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RootKey")
	ret0, _ := ret[0].(flow.AccountPrivateKey)
	return ret0
}

// RootKey indicates an expected call of RootKey
func (mr *MockEmulatedBlockchainAPIMockRecorder) RootKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RootKey", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).RootKey))
}

// SubmitTransaction mocks base method
func (m *MockEmulatedBlockchainAPI) SubmitTransaction(arg0 flow.Transaction) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubmitTransaction", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SubmitTransaction indicates an expected call of SubmitTransaction
func (mr *MockEmulatedBlockchainAPIMockRecorder) SubmitTransaction(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubmitTransaction", reflect.TypeOf((*MockEmulatedBlockchainAPI)(nil).SubmitTransaction), arg0)
}
