// Code generated by mockery v2.21.4. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"
)

// WALReader is an autogenerated mock type for the WALReader type
type WALReader struct {
	mock.Mock
}

// Next provides a mock function with given fields:
func (_m *WALReader) Next() (uint64, []flow.RegisterEntry, error) {
	ret := _m.Called()

	var r0 uint64
	var r1 []flow.RegisterEntry
	var r2 error
	if rf, ok := ret.Get(0).(func() (uint64, []flow.RegisterEntry, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func() []flow.RegisterEntry); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]flow.RegisterEntry)
		}
	}

	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

type mockConstructorTestingTNewWALReader interface {
	mock.TestingT
	Cleanup(func())
}

// NewWALReader creates a new instance of WALReader. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewWALReader(t mockConstructorTestingTNewWALReader) *WALReader {
	mock := &WALReader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
