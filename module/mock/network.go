// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import (
	mock "github.com/stretchr/testify/mock"

	network "github.com/dapperlabs/flow-go/network"
)

// Network is an autogenerated mock type for the Network type
type Network struct {
	mock.Mock
}

// Register provides a mock function with given fields: engineID, engine
func (_m *Network) Register(engineID uint8, engine network.Engine) (network.Conduit, error) {
	ret := _m.Called(engineID, engine)

	var r0 network.Conduit
	if rf, ok := ret.Get(0).(func(uint8, network.Engine) network.Conduit); ok {
		r0 = rf(engineID, engine)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(network.Conduit)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint8, network.Engine) error); ok {
		r1 = rf(engineID, engine)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
