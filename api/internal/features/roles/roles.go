// Package roles holds the role use cases: listing, creating, and assigning
// roles to users.
package roles

import (
	"context"

	"eventify/api/internal/domain"
	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
)

const columns = `id, name, description, created_at, updated_at`

func scanRoles(ctx context.Context, db postgres.Querier, sql string, args ...any) ([]domain.Role, error) {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.Internal, "query roles", err)
	}
	defer rows.Close()

	var out []domain.Role
	for rows.Next() {
		var r domain.Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, apperrors.Wrap(apperrors.Internal, "scan role", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListRolesHandler lists every role.
type ListRolesHandler struct{ db postgres.Querier }

func NewListRolesHandler(db postgres.Querier) ListRolesHandler { return ListRolesHandler{db: db} }

func (h ListRolesHandler) Handle(ctx context.Context) ([]domain.Role, error) {
	return scanRoles(ctx, h.db, `SELECT `+columns+` FROM roles ORDER BY name`)
}

// GetUserRolesQuery lists the roles assigned to a user.
type GetUserRolesQuery struct{ UserID uuid.UUID }

// GetUserRolesHandler lists a user's roles.
type GetUserRolesHandler struct{ db postgres.Querier }

func NewGetUserRolesHandler(db postgres.Querier) GetUserRolesHandler {
	return GetUserRolesHandler{db: db}
}

func (h GetUserRolesHandler) Handle(ctx context.Context, q GetUserRolesQuery) ([]domain.Role, error) {
	return scanRoles(ctx, h.db,
		`SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		   FROM roles r
		   JOIN user_roles ur ON ur.role_id = r.id
		  WHERE ur.user_id = $1
		  ORDER BY r.name`, q.UserID)
}
