// Code generated by mockery v1.0.0. DO NOT EDIT.

package rest

import context "context"
import mock "github.com/stretchr/testify/mock"

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, node, id
func (_m *MockClient) Create(ctx context.Context, node string, id string) (*Bundle, error) {
	ret := _m.Called(ctx, node, id)

	var r0 *Bundle
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *Bundle); ok {
		r0 = rf(ctx, node, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Bundle)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, node, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFile provides a mock function with given fields: ctx, node, id
func (_m *MockClient) GetFile(ctx context.Context, node string, id string) (string, error) {
	ret := _m.Called(ctx, node, id)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string, string) string); ok {
		r0 = rf(ctx, node, id)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, node, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Status provides a mock function with given fields: ctx, node, id
func (_m *MockClient) Status(ctx context.Context, node string, id string) (*Bundle, error) {
	ret := _m.Called(ctx, node, id)

	var r0 *Bundle
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *Bundle); ok {
		r0 = rf(ctx, node, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Bundle)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, node, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
