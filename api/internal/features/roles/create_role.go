package roles

import (
	"context"
	"errors"
	"time"

	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolation = "23505"

// CreateRoleCommand creates a role.
type CreateRoleCommand struct {
	Name        string
	Description string
}

// CreateRoleResult identifies the created role.
type CreateRoleResult struct {
	CreatedAt time.Time
	RoleID    uuid.UUID
}

// CreateRoleHandler creates a role.
type CreateRoleHandler struct{ db postgres.Querier }

func NewCreateRoleHandler(db postgres.Querier) CreateRoleHandler { return CreateRoleHandler{db: db} }

func (h CreateRoleHandler) Handle(ctx context.Context, cmd CreateRoleCommand) (CreateRoleResult, error) {
	var res CreateRoleResult
	if cmd.Name == "" {
		return res, apperrors.New(apperrors.Invalid, "role name is required")
	}

	err := h.db.QueryRow(ctx,
		`INSERT INTO roles (id, name, description, created_at)
		 VALUES ($1, $2, $3, now())
		 RETURNING id, created_at`,
		uuid.New(), cmd.Name, cmd.Description,
	).Scan(&res.RoleID, &res.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return res, apperrors.New(apperrors.Conflict, "a role with this name already exists")
	}
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "insert role", err)
	}
	return res, nil
}
