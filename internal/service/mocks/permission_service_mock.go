// Code generated by MockGen. DO NOT EDIT.
// Source: internal/service/permission_service.go

// Package mocks is a generated GoMock package.
package mocks

import (
	domain "eventify/internal/domain"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	uuid "github.com/google/uuid"
)

// MockIPermissionService is a mock of IPermissionService interface.
type MockIPermissionService struct {
	ctrl     *gomock.Controller
	recorder *MockIPermissionServiceMockRecorder
}

// MockIPermissionServiceMockRecorder is the mock recorder for MockIPermissionService.
type MockIPermissionServiceMockRecorder struct {
	mock *MockIPermissionService
}

// NewMockIPermissionService creates a new mock instance.
func NewMockIPermissionService(ctrl *gomock.Controller) *MockIPermissionService {
	mock := &MockIPermissionService{ctrl: ctrl}
	mock.recorder = &MockIPermissionServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIPermissionService) EXPECT() *MockIPermissionServiceMockRecorder {
	return m.recorder
}

// AssignPermissionToRole mocks base method.
func (m *MockIPermissionService) AssignPermissionToRole(roleID, permissionID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AssignPermissionToRole", roleID, permissionID)
	ret0, _ := ret[0].(error)
	return ret0
}

// AssignPermissionToRole indicates an expected call of AssignPermissionToRole.
func (mr *MockIPermissionServiceMockRecorder) AssignPermissionToRole(roleID, permissionID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AssignPermissionToRole", reflect.TypeOf((*MockIPermissionService)(nil).AssignPermissionToRole), roleID, permissionID)
}

// Create mocks base method.
func (m *MockIPermissionService) Create(permission *domain.Permission) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", permission)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockIPermissionServiceMockRecorder) Create(permission interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockIPermissionService)(nil).Create), permission)
}

// GetAll mocks base method.
func (m *MockIPermissionService) GetAll() ([]domain.Permission, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll")
	ret0, _ := ret[0].([]domain.Permission)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAll indicates an expected call of GetAll.
func (mr *MockIPermissionServiceMockRecorder) GetAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockIPermissionService)(nil).GetAll))
}

// GetPermissions mocks base method.
func (m *MockIPermissionService) GetPermissions(userID uuid.UUID) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPermissions", userID)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPermissions indicates an expected call of GetPermissions.
func (mr *MockIPermissionServiceMockRecorder) GetPermissions(userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPermissions", reflect.TypeOf((*MockIPermissionService)(nil).GetPermissions), userID)
}

// GetRolePermissions mocks base method.
func (m *MockIPermissionService) GetRolePermissions(roleID uuid.UUID) ([]domain.Permission, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRolePermissions", roleID)
	ret0, _ := ret[0].([]domain.Permission)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRolePermissions indicates an expected call of GetRolePermissions.
func (mr *MockIPermissionServiceMockRecorder) GetRolePermissions(roleID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRolePermissions", reflect.TypeOf((*MockIPermissionService)(nil).GetRolePermissions), roleID)
}

// RemovePermissionFromRole mocks base method.
func (m *MockIPermissionService) RemovePermissionFromRole(roleID, permissionID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemovePermissionFromRole", roleID, permissionID)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemovePermissionFromRole indicates an expected call of RemovePermissionFromRole.
func (mr *MockIPermissionServiceMockRecorder) RemovePermissionFromRole(roleID, permissionID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemovePermissionFromRole", reflect.TypeOf((*MockIPermissionService)(nil).RemovePermissionFromRole), roleID, permissionID)
}
