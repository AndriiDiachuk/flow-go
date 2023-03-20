// Code generated by mockery v2.21.4. DO NOT EDIT.

package mockp2p

import (
	context "context"

	host "github.com/libp2p/go-libp2p/core/host"
	mock "github.com/stretchr/testify/mock"

	p2p "github.com/onflow/flow-go/network/p2p"

	zerolog "github.com/rs/zerolog"
)

// GossipSubFactoryFunc is an autogenerated mock type for the GossipSubFactoryFunc type
type GossipSubFactoryFunc struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0, _a1, _a2, _a3
func (_m *GossipSubFactoryFunc) Execute(_a0 context.Context, _a1 zerolog.Logger, _a2 host.Host, _a3 p2p.PubSubAdapterConfig) (p2p.PubSubAdapter, error) {
	ret := _m.Called(_a0, _a1, _a2, _a3)

	var r0 p2p.PubSubAdapter
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, zerolog.Logger, host.Host, p2p.PubSubAdapterConfig) (p2p.PubSubAdapter, error)); ok {
		return rf(_a0, _a1, _a2, _a3)
	}
	if rf, ok := ret.Get(0).(func(context.Context, zerolog.Logger, host.Host, p2p.PubSubAdapterConfig) p2p.PubSubAdapter); ok {
		r0 = rf(_a0, _a1, _a2, _a3)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(p2p.PubSubAdapter)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, zerolog.Logger, host.Host, p2p.PubSubAdapterConfig) error); ok {
		r1 = rf(_a0, _a1, _a2, _a3)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewGossipSubFactoryFunc interface {
	mock.TestingT
	Cleanup(func())
}

// NewGossipSubFactoryFunc creates a new instance of GossipSubFactoryFunc. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewGossipSubFactoryFunc(t mockConstructorTestingTNewGossipSubFactoryFunc) *GossipSubFactoryFunc {
	mock := &GossipSubFactoryFunc{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
