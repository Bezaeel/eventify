package service

import (
	"eventify/internal/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleService struct {
	db *gorm.DB
}

type IRoleService interface {
	GetAll() ([]domain.Role, error)
	GetByID(id uuid.UUID) (*domain.Role, error)
	Create(role *domain.Role) error
	AssignRoleToUser(userID, roleID uuid.UUID) error
	RemoveRoleFromUser(userID, roleID uuid.UUID) error
	GetUserRoles(userID uuid.UUID) ([]domain.Role, error)
}

func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{db}
}

// GetAll retrieves all roles from the database.
func (r *RoleService) GetAll() ([]domain.Role, error) {
	var roles []domain.Role
	if err := r.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetByID retrieves a role by its ID.
func (r *RoleService) GetByID(id uuid.UUID) (*domain.Role, error) {
	var role domain.Role
	if err := r.db.First(&role, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// Create creates a new role in the database.
func (r *RoleService) Create(role *domain.Role) error {
	if role.Id == uuid.Nil {
		role.Id = uuid.New()
	}

	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = &now

	if err := r.db.Create(role).Error; err != nil {
		return err
	}
	return nil
}

// AssignRoleToUser assigns a role to a user.
func (r *RoleService) AssignRoleToUser(userID, roleID uuid.UUID) error {
	userRole := domain.UserRole{
		UserId: userID,
		RoleId: roleID,
	}
	if err := r.db.Create(&userRole).Error; err != nil {
		return err
	}
	return nil
}

// RemoveRoleFromUser removes a role from a user.
func (r *RoleService) RemoveRoleFromUser(userID, roleID uuid.UUID) error {
	if err := r.db.Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&domain.UserRole{}).Error; err != nil {
		return err
	}
	return nil
}

// GetUserRoles retrieves all roles assigned to a user.
func (r *RoleService) GetUserRoles(userID uuid.UUID) ([]domain.Role, error) {
	var roles []domain.Role
	if err := r.db.Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
