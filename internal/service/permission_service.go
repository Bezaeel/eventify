package service

import (
	"eventify/internal/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionService struct {
	db *gorm.DB
}

type IPermissionService interface {
	GetAll() ([]domain.Permission, error)
	GetPermissions(userID uuid.UUID) ([]string, error)
	Create(permission *domain.Permission) error
	AssignPermissionToRole(roleID, permissionID uuid.UUID) error
	RemovePermissionFromRole(roleID, permissionID uuid.UUID) error
	GetRolePermissions(roleID uuid.UUID) ([]domain.Permission, error)
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db}
}

func (p *PermissionService) GetAll() ([]domain.Permission, error) {
	var permissions []domain.Permission
	if err := p.db.Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetPermissions retrieves permissions for a user from the database
func (p *PermissionService) GetPermissions(userID uuid.UUID) ([]string, error) {
	var permissions []string

	// Use GORM's Joins method to query permissions via roles
	err := p.db.Table("permissions").
		Select("permissions.name").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Joins("JOIN user_roles ON role_permissions.role_id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Scan(&permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}

func (p *PermissionService) Create(permission *domain.Permission) error {
	if permission.Id == uuid.Nil {
		permission.Id = uuid.New()
	}

	now := time.Now()
	permission.CreatedAt = now
	permission.UpdatedAt = &now

	if err := p.db.Create(permission).Error; err != nil {
		return err
	}
	return nil
}

func (p *PermissionService) AssignPermissionToRole(roleID, permissionID uuid.UUID) error {
	rolePermission := domain.RolePermissions{
		RoleId:       roleID,
		PermissionId: permissionID,
	}
	if err := p.db.Create(&rolePermission).Error; err != nil {
		return err
	}
	return nil
}

func (p *PermissionService) RemovePermissionFromRole(roleID, permissionID uuid.UUID) error {
	if err := p.db.Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&domain.RolePermissions{}).Error; err != nil {
		return err
	}
	return nil
}

func (p *PermissionService) GetRolePermissions(roleID uuid.UUID) ([]domain.Permission, error) {
	var permissions []domain.Permission
	if err := p.db.Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}
