// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"
)

// MatchingCore is an autogenerated mock type for the MatchingCore type
type MatchingCore struct {
	mock.Mock
}

// ProcessFinalizedBlock provides a mock function with given fields: finalizedBlockID
func (_m *MatchingCore) ProcessFinalizedBlock(finalizedBlockID flow.Identifier) error {
	ret := _m.Called(finalizedBlockID)

	var r0 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) error); ok {
		r0 = rf(finalizedBlockID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ProcessReceipt provides a mock function with given fields: receipt
func (_m *MatchingCore) ProcessReceipt(receipt *flow.ExecutionReceipt) error {
	ret := _m.Called(receipt)

	var r0 error
	if rf, ok := ret.Get(0).(func(*flow.ExecutionReceipt) error); ok {
		r0 = rf(receipt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
