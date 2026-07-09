package roles

import (
	"context"
	"errors"

	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const foreignKeyViolation = "23503"

// AssignRoleCommand grants a role to a user.
type AssignRoleCommand struct {
	UserID uuid.UUID
	RoleID uuid.UUID
}

// AssignRoleHandler grants a role.
type AssignRoleHandler struct{ db postgres.Querier }

func NewAssignRoleHandler(db postgres.Querier) AssignRoleHandler { return AssignRoleHandler{db: db} }

// Handle grants the role, tolerating a repeat grant.
//
// ON CONFLICT DO NOTHING makes this idempotent. The old AssignRoleToUser did a
// bare INSERT, so granting the same role twice surfaced a driver error as a
// 500. A missing user or role is a Postgres foreign-key violation, which maps
// to Invalid rather than a 500.
func (h AssignRoleHandler) Handle(ctx context.Context, cmd AssignRoleCommand) error {
	_, err := h.db.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		cmd.UserID, cmd.RoleID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return apperrors.New(apperrors.Invalid, "user or role does not exist")
	}
	if err != nil {
		return apperrors.Wrap(apperrors.Internal, "assign role", err)
	}
	return nil
}

// RemoveRoleCommand revokes a role from a user.
type RemoveRoleCommand struct {
	UserID uuid.UUID
	RoleID uuid.UUID
}

// RemoveRoleHandler revokes a role.
type RemoveRoleHandler struct{ db postgres.Querier }

func NewRemoveRoleHandler(db postgres.Querier) RemoveRoleHandler { return RemoveRoleHandler{db: db} }

// Handle revokes the role, reporting NotFound when the grant did not exist.
func (h RemoveRoleHandler) Handle(ctx context.Context, cmd RemoveRoleCommand) error {
	tag, err := h.db.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, cmd.UserID, cmd.RoleID)
	if err != nil {
		return apperrors.Wrap(apperrors.Internal, "remove role", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.New(apperrors.NotFound, "user does not have this role")
	}
	return nil
}
