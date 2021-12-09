// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	flow "github.com/onflow/flow-go/model/flow"

	mock "github.com/stretchr/testify/mock"

	model "github.com/onflow/flow-go/consensus/hotstuff/model"
)

// Verifier is an autogenerated mock type for the Verifier type
type Verifier struct {
	mock.Mock
}

// VerifyQC provides a mock function with given fields: voters, sigData, block
func (_m *Verifier) VerifyQC(voters flow.IdentityList, sigData []byte, block *model.Block) error {
	ret := _m.Called(voters, sigData, block)

	var r0 error
	if rf, ok := ret.Get(0).(func(flow.IdentityList, []byte, *model.Block) error); ok {
		r0 = rf(voters, sigData, block)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// VerifyVote provides a mock function with given fields: voter, sigData, block
func (_m *Verifier) VerifyVote(voter *flow.Identity, sigData []byte, block *model.Block) error {
	ret := _m.Called(voter, sigData, block)

	var r0 error
	if rf, ok := ret.Get(0).(func(*flow.Identity, []byte, *model.Block) error); ok {
		r0 = rf(voter, sigData, block)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
