package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	Id              uuid.UUID         `json:"id" gorm:"primaryKey;column:id"`
	Name            string            `json:"name" gorm:"column:name"`
	Description     string            `json:"description" gorm:"column:description"`
	RolePermissions []RolePermissions `json:"permissions" gorm:"foreignKey:RoleId;references:Id"`
	UserRoles       []UserRole        `json:"user_roles,omitempty" gorm:"foreignKey:RoleId;references:Id"`
	CreatedAt       time.Time         `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       *time.Time         `json:"updated_at" gorm:"column:updated_at"`
}

type IRoleService interface {
	Create(role *Role) error
	GetByID(id uuid.UUID) (*Role, error)
	GetAllRoles() ([]Role, error)
	Update(role *Role) error
	Delete(id uuid.UUID) error
}
