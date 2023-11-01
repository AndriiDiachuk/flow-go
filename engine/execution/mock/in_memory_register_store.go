// Code generated by mockery v2.21.4. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"
)

// InMemoryRegisterStore is an autogenerated mock type for the InMemoryRegisterStore type
type InMemoryRegisterStore struct {
	mock.Mock
}

// GetRegister provides a mock function with given fields: height, blockID, register
func (_m *InMemoryRegisterStore) GetRegister(height uint64, blockID flow.Identifier, register flow.RegisterID) ([]byte, error) {
	ret := _m.Called(height, blockID, register)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, flow.Identifier, flow.RegisterID) ([]byte, error)); ok {
		return rf(height, blockID, register)
	}
	if rf, ok := ret.Get(0).(func(uint64, flow.Identifier, flow.RegisterID) []byte); ok {
		r0 = rf(height, blockID, register)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, flow.Identifier, flow.RegisterID) error); ok {
		r1 = rf(height, blockID, register)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUpdatedRegisters provides a mock function with given fields: height, blockID
func (_m *InMemoryRegisterStore) GetUpdatedRegisters(height uint64, blockID flow.Identifier) ([]flow.RegisterEntry, error) {
	ret := _m.Called(height, blockID)

	var r0 []flow.RegisterEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, flow.Identifier) ([]flow.RegisterEntry, error)); ok {
		return rf(height, blockID)
	}
	if rf, ok := ret.Get(0).(func(uint64, flow.Identifier) []flow.RegisterEntry); ok {
		r0 = rf(height, blockID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]flow.RegisterEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, flow.Identifier) error); ok {
		r1 = rf(height, blockID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsBlockExecuted provides a mock function with given fields: height, blockID
func (_m *InMemoryRegisterStore) IsBlockExecuted(height uint64, blockID flow.Identifier) (bool, error) {
	ret := _m.Called(height, blockID)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, flow.Identifier) (bool, error)); ok {
		return rf(height, blockID)
	}
	if rf, ok := ret.Get(0).(func(uint64, flow.Identifier) bool); ok {
		r0 = rf(height, blockID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(uint64, flow.Identifier) error); ok {
		r1 = rf(height, blockID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Prune provides a mock function with given fields: finalizedHeight, finalizedBlockID
func (_m *InMemoryRegisterStore) Prune(finalizedHeight uint64, finalizedBlockID flow.Identifier) error {
	ret := _m.Called(finalizedHeight, finalizedBlockID)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64, flow.Identifier) error); ok {
		r0 = rf(finalizedHeight, finalizedBlockID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PrunedHeight provides a mock function with given fields:
func (_m *InMemoryRegisterStore) PrunedHeight() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// SaveRegisters provides a mock function with given fields: height, blockID, parentID, registers
func (_m *InMemoryRegisterStore) SaveRegisters(height uint64, blockID flow.Identifier, parentID flow.Identifier, registers []flow.RegisterEntry) error {
	ret := _m.Called(height, blockID, parentID, registers)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64, flow.Identifier, flow.Identifier, []flow.RegisterEntry) error); ok {
		r0 = rf(height, blockID, parentID, registers)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewInMemoryRegisterStore interface {
	mock.TestingT
	Cleanup(func())
}

// NewInMemoryRegisterStore creates a new instance of InMemoryRegisterStore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewInMemoryRegisterStore(t mockConstructorTestingTNewInMemoryRegisterStore) *InMemoryRegisterStore {
	mock := &InMemoryRegisterStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}