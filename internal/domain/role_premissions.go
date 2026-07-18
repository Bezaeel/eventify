package domain

import (
	"time"

	"github.com/google/uuid"
)

type RolePermissions struct {
	RoleId       uuid.UUID  `json:"role_id" gorm:"primaryKey;column:role_id"`
	PermissionId uuid.UUID  `json:"permission_id" gorm:"primaryKey;column:permission_id"`
	Role         Role       `json:"role" gorm:"foreignKey:RoleId;references:Id"`
	Permission   Permission `json:"permission" gorm:"foreignKey:PermissionId;references:Id"`
	CreatedAt    time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    *time.Time  `json:"updated_at" gorm:"column:updated_at"`
}
