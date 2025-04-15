package domain

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	Id              uuid.UUID         `json:"id" gorm:"primaryKey;column:id"`
	Name            string            `json:"name" gorm:"column:name"`
	Description     string            `json:"description" gorm:"column:description"`
	RolePermissions []RolePermissions `json:"role_permissions" gorm:"foreignKey:PermissionId;references:Id"`
	CreatedAt       time.Time         `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time         `json:"updated_at" gorm:"column:updated_at"`
}

type IPermissionService interface {
	Create(permission *Permission) error
	GetByID(id uuid.UUID) (*Permission, error)
	GetAllPermissions() ([]Permission, error)
	Update(permission *Permission) error
	Delete(id uuid.UUID) error
}
