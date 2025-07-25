// Code generated by MockGen. DO NOT EDIT.
// Source: internal/service/role_service.go

// Package mocks is a generated GoMock package.
package mocks

import (
	domain "eventify/internal/domain"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	uuid "github.com/google/uuid"
)

// MockIRoleService is a mock of IRoleService interface.
type MockIRoleService struct {
	ctrl     *gomock.Controller
	recorder *MockIRoleServiceMockRecorder
}

// MockIRoleServiceMockRecorder is the mock recorder for MockIRoleService.
type MockIRoleServiceMockRecorder struct {
	mock *MockIRoleService
}

// NewMockIRoleService creates a new mock instance.
func NewMockIRoleService(ctrl *gomock.Controller) *MockIRoleService {
	mock := &MockIRoleService{ctrl: ctrl}
	mock.recorder = &MockIRoleServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIRoleService) EXPECT() *MockIRoleServiceMockRecorder {
	return m.recorder
}

// AssignRoleToUser mocks base method.
func (m *MockIRoleService) AssignRoleToUser(userID, roleID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AssignRoleToUser", userID, roleID)
	ret0, _ := ret[0].(error)
	return ret0
}

// AssignRoleToUser indicates an expected call of AssignRoleToUser.
func (mr *MockIRoleServiceMockRecorder) AssignRoleToUser(userID, roleID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AssignRoleToUser", reflect.TypeOf((*MockIRoleService)(nil).AssignRoleToUser), userID, roleID)
}

// Create mocks base method.
func (m *MockIRoleService) Create(role *domain.Role) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", role)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockIRoleServiceMockRecorder) Create(role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockIRoleService)(nil).Create), role)
}

// GetAll mocks base method.
func (m *MockIRoleService) GetAll() ([]domain.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll")
	ret0, _ := ret[0].([]domain.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAll indicates an expected call of GetAll.
func (mr *MockIRoleServiceMockRecorder) GetAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockIRoleService)(nil).GetAll))
}

// GetByID mocks base method.
func (m *MockIRoleService) GetByID(id uuid.UUID) (*domain.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", id)
	ret0, _ := ret[0].(*domain.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockIRoleServiceMockRecorder) GetByID(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockIRoleService)(nil).GetByID), id)
}

// GetUserRoles mocks base method.
func (m *MockIRoleService) GetUserRoles(userID uuid.UUID) ([]domain.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserRoles", userID)
	ret0, _ := ret[0].([]domain.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserRoles indicates an expected call of GetUserRoles.
func (mr *MockIRoleServiceMockRecorder) GetUserRoles(userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserRoles", reflect.TypeOf((*MockIRoleService)(nil).GetUserRoles), userID)
}

// RemoveRoleFromUser mocks base method.
func (m *MockIRoleService) RemoveRoleFromUser(userID, roleID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveRoleFromUser", userID, roleID)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveRoleFromUser indicates an expected call of RemoveRoleFromUser.
func (mr *MockIRoleServiceMockRecorder) RemoveRoleFromUser(userID, roleID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveRoleFromUser", reflect.TypeOf((*MockIRoleService)(nil).RemoveRoleFromUser), userID, roleID)
}
