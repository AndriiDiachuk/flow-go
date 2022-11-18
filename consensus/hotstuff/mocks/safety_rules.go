// Code generated by mockery v2.13.1. DO NOT EDIT.

package mocks

import (
	flow "github.com/onflow/flow-go/model/flow"

	mock "github.com/stretchr/testify/mock"

	model "github.com/onflow/flow-go/consensus/hotstuff/model"
)

// SafetyRules is an autogenerated mock type for the SafetyRules type
type SafetyRules struct {
	mock.Mock
}

// ProduceTimeout provides a mock function with given fields: curView, newestQC, lastViewTC
func (_m *SafetyRules) ProduceTimeout(curView uint64, newestQC *flow.QuorumCertificate, lastViewTC *flow.TimeoutCertificate) (*model.TimeoutObject, error) {
	ret := _m.Called(curView, newestQC, lastViewTC)

	var r0 *model.TimeoutObject
	if rf, ok := ret.Get(0).(func(uint64, *flow.QuorumCertificate, *flow.TimeoutCertificate) *model.TimeoutObject); ok {
		r0 = rf(curView, newestQC, lastViewTC)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.TimeoutObject)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint64, *flow.QuorumCertificate, *flow.TimeoutCertificate) error); ok {
		r1 = rf(curView, newestQC, lastViewTC)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ProduceVote provides a mock function with given fields: proposal, curView
func (_m *SafetyRules) ProduceVote(proposal *model.Proposal, curView uint64) (*model.Vote, error) {
	ret := _m.Called(proposal, curView)

	var r0 *model.Vote
	if rf, ok := ret.Get(0).(func(*model.Proposal, uint64) *model.Vote); ok {
		r0 = rf(proposal, curView)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Vote)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.Proposal, uint64) error); ok {
		r1 = rf(proposal, curView)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewSafetyRules interface {
	mock.TestingT
	Cleanup(func())
}

// NewSafetyRules creates a new instance of SafetyRules. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSafetyRules(t mockConstructorTestingTNewSafetyRules) *SafetyRules {
	mock := &SafetyRules{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}