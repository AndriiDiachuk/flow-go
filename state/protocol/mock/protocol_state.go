// Code generated by mockery v2.21.4. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"

	protocol "github.com/onflow/flow-go/state/protocol"
)

// ProtocolState is an autogenerated mock type for the ProtocolState type
type ProtocolState struct {
	mock.Mock
}

// AtBlockID provides a mock function with given fields: blockID
func (_m *ProtocolState) AtBlockID(blockID flow.Identifier) (protocol.DynamicProtocolState, error) {
	ret := _m.Called(blockID)

	var r0 protocol.DynamicProtocolState
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) (protocol.DynamicProtocolState, error)); ok {
		return rf(blockID)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier) protocol.DynamicProtocolState); ok {
		r0 = rf(blockID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(protocol.DynamicProtocolState)
		}
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(blockID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GlobalParams provides a mock function with given fields:
func (_m *ProtocolState) GlobalParams() protocol.GlobalParams {
	ret := _m.Called()

	var r0 protocol.GlobalParams
	if rf, ok := ret.Get(0).(func() protocol.GlobalParams); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(protocol.GlobalParams)
		}
	}

	return r0
}

// KVStoreAtBlockID provides a mock function with given fields: blockID
func (_m *ProtocolState) KVStoreAtBlockID(blockID flow.Identifier) (protocol.KVStoreReader, error) {
	ret := _m.Called(blockID)

	var r0 protocol.KVStoreReader
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) (protocol.KVStoreReader, error)); ok {
		return rf(blockID)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier) protocol.KVStoreReader); ok {
		r0 = rf(blockID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(protocol.KVStoreReader)
		}
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(blockID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewProtocolState interface {
	mock.TestingT
	Cleanup(func())
}

// NewProtocolState creates a new instance of ProtocolState. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewProtocolState(t mockConstructorTestingTNewProtocolState) *ProtocolState {
	mock := &ProtocolState{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
