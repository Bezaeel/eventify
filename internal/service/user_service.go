package service

import (
	"errors"
	"eventify/internal/domain"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

type IUserService interface {
	Create(user *domain.User) error
	GetByID(id uuid.UUID) (*domain.User, error)
	GetByEmail(email string) (*domain.User, error)
	Update(user *domain.User) error
	UpdatePassword(id uuid.UUID, password string) error
	Delete(id uuid.UUID) error
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db}
}

func (ur *UserService) Create(user *domain.User) error {
	// Generate new UUID if not provided
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Use GORM's Exec method for raw SQL
	err = ur.db.Exec(
		"INSERT INTO users (id, email, password, first_name, last_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		user.ID, user.Email, string(hashedPassword), user.FirstName, user.LastName, user.CreatedAt, user.UpdatedAt,
	).Error
	return err
}

func (ur *UserService) GetByID(id uuid.UUID) (*domain.User, error) {
	user := &domain.User{}
	// Use GORM's Raw method for raw SQL
	err := ur.db.Raw(
		"SELECT id, email, password, first_name, last_name, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return user, nil
}

func (ur *UserService) GetByEmail(email string) (*domain.User, error) {
	user := &domain.User{}
	// Use GORM's Raw method for raw SQL
	err := ur.db.Raw(
		"SELECT id, email, password, first_name, last_name, created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return user, nil
}

func (ur *UserService) Update(user *domain.User) error {
	user.UpdatedAt = time.Now()

	// Use GORM's Exec method for raw SQL
	err := ur.db.Exec(
		"UPDATE users SET email = ?, first_name = ?, last_name = ?, updated_at = ? WHERE id = ?",
		user.Email, user.FirstName, user.LastName, user.UpdatedAt, user.ID,
	).Error
	return err
}

func (ur *UserService) UpdatePassword(id uuid.UUID, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Use GORM's Exec method for raw SQL
	err = ur.db.Exec(
		"UPDATE users SET password = ?, updated_at = ? WHERE id = ?",
		string(hashedPassword), time.Now(), id,
	).Error
	return err
}

func (ur *UserService) Delete(id uuid.UUID) error {
	// Use GORM's Exec method for raw SQL
	err := ur.db.Exec("DELETE FROM users WHERE id = ?", id).Error
	return err
}

func (ur *UserService) CheckPassword(email, password string) (*domain.User, error) {
	user, err := ur.GetByEmail(email)
	if err != nil {
		return nil, err
	}

	// Compare the hashed password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return user, nil
}
