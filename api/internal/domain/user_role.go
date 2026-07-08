package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole struct {
	UserId    uuid.UUID `json:"user_id" gorm:"primaryKey;column:user_id"`
	RoleId    uuid.UUID `json:"role_id" gorm:"primaryKey;column:role_id"`
	User      User      `json:"user" gorm:"foreignKey:UserId;references:ID"`
	Role      Role      `json:"role" gorm:"foreignKey:RoleId;references:Id"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`
}
