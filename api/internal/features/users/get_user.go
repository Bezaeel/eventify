package users

import (
	"context"
	"errors"

	"eventify/api/internal/domain"
	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetUserQuery fetches a user by id.
type GetUserQuery struct {
	UserID uuid.UUID
}

// GetUserHandler fetches a user.
type GetUserHandler struct {
	db postgres.Querier
}

func NewGetUserHandler(db postgres.Querier) GetUserHandler {
	return GetUserHandler{db: db}
}

// Handle returns the user, or NotFound.
func (h GetUserHandler) Handle(ctx context.Context, q GetUserQuery) (domain.User, error) {
	u, err := scanUser(h.db.QueryRow(ctx, `SELECT `+columns+` FROM users WHERE id = $1`, q.UserID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, apperrors.New(apperrors.NotFound, "user not found")
	}
	if err != nil {
		return domain.User{}, apperrors.Wrap(apperrors.Internal, "get user", err)
	}
	return u, nil
}

// GetUserPermissionsQuery lists a user's effective permission names.
type GetUserPermissionsQuery struct {
	UserID uuid.UUID
}

// GetUserPermissionsHandler resolves permissions through roles.
type GetUserPermissionsHandler struct {
	db postgres.Querier
}

func NewGetUserPermissionsHandler(db postgres.Querier) GetUserPermissionsHandler {
	return GetUserPermissionsHandler{db: db}
}

// Handle returns the permission names granted via the user's roles.
func (h GetUserPermissionsHandler) Handle(ctx context.Context, q GetUserPermissionsQuery) ([]string, error) {
	return permissionsFor(ctx, h.db, q.UserID)
}
