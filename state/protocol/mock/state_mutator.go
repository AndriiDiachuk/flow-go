// Code generated by mockery v2.21.4. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"

	transaction "github.com/onflow/flow-go/storage/badger/transaction"
)

// StateMutator is an autogenerated mock type for the StateMutator type
type StateMutator struct {
	mock.Mock
}

// ApplyServiceEventsFromValidatedSeals provides a mock function with given fields: seals
func (_m *StateMutator) ApplyServiceEventsFromValidatedSeals(seals []*flow.Seal) error {
	ret := _m.Called(seals)

	var r0 error
	if rf, ok := ret.Get(0).(func([]*flow.Seal) error); ok {
		r0 = rf(seals)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Build provides a mock function with given fields:
func (_m *StateMutator) Build() (bool, *flow.ProtocolStateEntry, flow.Identifier, []func(*transaction.Tx) error) {
	ret := _m.Called()

	var r0 bool
	var r1 *flow.ProtocolStateEntry
	var r2 flow.Identifier
	var r3 []func(*transaction.Tx) error
	if rf, ok := ret.Get(0).(func() (bool, *flow.ProtocolStateEntry, flow.Identifier, []func(*transaction.Tx) error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() *flow.ProtocolStateEntry); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*flow.ProtocolStateEntry)
		}
	}

	if rf, ok := ret.Get(2).(func() flow.Identifier); ok {
		r2 = rf()
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).(flow.Identifier)
		}
	}

	if rf, ok := ret.Get(3).(func() []func(*transaction.Tx) error); ok {
		r3 = rf()
	} else {
		if ret.Get(3) != nil {
			r3 = ret.Get(3).([]func(*transaction.Tx) error)
		}
	}

	return r0, r1, r2, r3
}

type mockConstructorTestingTNewStateMutator interface {
	mock.TestingT
	Cleanup(func())
}

// NewStateMutator creates a new instance of StateMutator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewStateMutator(t mockConstructorTestingTNewStateMutator) *StateMutator {
	mock := &StateMutator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}