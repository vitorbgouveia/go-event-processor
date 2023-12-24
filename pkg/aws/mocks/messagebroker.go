// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/aws/messagebroker.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	aws "github.com/vitorbgouveia/go-event-processor/pkg/aws"
)

// MockMessageBroker is a mock of MessageBroker interface.
type MockMessageBroker struct {
	ctrl     *gomock.Controller
	recorder *MockMessageBrokerMockRecorder
}

// MockMessageBrokerMockRecorder is the mock recorder for MockMessageBroker.
type MockMessageBrokerMockRecorder struct {
	mock *MockMessageBroker
}

// NewMockMessageBroker creates a new mock instance.
func NewMockMessageBroker(ctrl *gomock.Controller) *MockMessageBroker {
	mock := &MockMessageBroker{ctrl: ctrl}
	mock.recorder = &MockMessageBrokerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMessageBroker) EXPECT() *MockMessageBrokerMockRecorder {
	return m.recorder
}

// CreateRejectEventTopic mocks base method.
func (m *MockMessageBroker) CreateRejectEventTopic(name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRejectEventTopic", name)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateRejectEventTopic indicates an expected call of CreateRejectEventTopic.
func (mr *MockMessageBrokerMockRecorder) CreateRejectEventTopic(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRejectEventTopic", reflect.TypeOf((*MockMessageBroker)(nil).CreateRejectEventTopic), name)
}

// CreateValidEventTopic mocks base method.
func (m *MockMessageBroker) CreateValidEventTopic(name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateValidEventTopic", name)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateValidEventTopic indicates an expected call of CreateValidEventTopic.
func (mr *MockMessageBrokerMockRecorder) CreateValidEventTopic(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateValidEventTopic", reflect.TypeOf((*MockMessageBroker)(nil).CreateValidEventTopic), name)
}

// PublishRejectedEvent mocks base method.
func (m *MockMessageBroker) PublishRejectedEvent(ctx context.Context, message string, errReason error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishRejectedEvent", ctx, message, errReason)
	ret0, _ := ret[0].(error)
	return ret0
}

// PublishRejectedEvent indicates an expected call of PublishRejectedEvent.
func (mr *MockMessageBrokerMockRecorder) PublishRejectedEvent(ctx, message, errReason interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishRejectedEvent", reflect.TypeOf((*MockMessageBroker)(nil).PublishRejectedEvent), ctx, message, errReason)
}

// PublishValidEvent mocks base method.
func (m *MockMessageBroker) PublishValidEvent(ctx context.Context, message string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishValidEvent", ctx, message)
	ret0, _ := ret[0].(error)
	return ret0
}

// PublishValidEvent indicates an expected call of PublishValidEvent.
func (mr *MockMessageBrokerMockRecorder) PublishValidEvent(ctx, message interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishValidEvent", reflect.TypeOf((*MockMessageBroker)(nil).PublishValidEvent), ctx, message)
}

// SendToQueue mocks base method.
func (m *MockMessageBroker) SendToQueue(ctx context.Context, p aws.SendToQueueInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendToQueue", ctx, p)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendToQueue indicates an expected call of SendToQueue.
func (mr *MockMessageBrokerMockRecorder) SendToQueue(ctx, p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendToQueue", reflect.TypeOf((*MockMessageBroker)(nil).SendToQueue), ctx, p)
}
