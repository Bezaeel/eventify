package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;column:id"`
	Email     string    `json:"email" gorm:"column:email;uniqueIndex"`
	Password  string    `json:"-" gorm:"column:password"` // Don't expose the password in JSON
	FirstName string    `json:"first_name" gorm:"column:first_name"`
	LastName  string    `json:"last_name" gorm:"column:last_name"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
	// Add UserRoles navigation property
	UserRoles []UserRole `json:"user_roles,omitempty" gorm:"foreignKey:UserId;references:ID"`
}

// UserService defines operations on users
type IUserService interface {
	Create(user *User) error
	GetByID(id uuid.UUID) (*User, error)
	GetByEmail(email string) (*User, error)
	GetAllUsers() ([]User, error)
	Update(user *User) error
	Delete(id uuid.UUID) error
}
