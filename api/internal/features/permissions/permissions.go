// Package permissions holds the permission use cases: listing permissions and
// binding them to roles.
package permissions

import (
	"context"
	"errors"

	"eventify/api/internal/domain"
	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const foreignKeyViolation = "23503"

const columns = `id, name, description, created_at, updated_at`

func scanPermissions(ctx context.Context, db postgres.Querier, sql string, args ...any) ([]domain.Permission, error) {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.Internal, "query permissions", err)
	}
	defer rows.Close()

	var out []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, apperrors.Wrap(apperrors.Internal, "scan permission", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListPermissionsHandler lists every permission.
type ListPermissionsHandler struct{ db postgres.Querier }

func NewListPermissionsHandler(db postgres.Querier) ListPermissionsHandler {
	return ListPermissionsHandler{db: db}
}

func (h ListPermissionsHandler) Handle(ctx context.Context) ([]domain.Permission, error) {
	return scanPermissions(ctx, h.db, `SELECT `+columns+` FROM permissions ORDER BY name`)
}

// GetRolePermissionsQuery lists the permissions bound to a role.
type GetRolePermissionsQuery struct{ RoleID uuid.UUID }

// GetRolePermissionsHandler lists a role's permissions.
type GetRolePermissionsHandler struct{ db postgres.Querier }

func NewGetRolePermissionsHandler(db postgres.Querier) GetRolePermissionsHandler {
	return GetRolePermissionsHandler{db: db}
}

func (h GetRolePermissionsHandler) Handle(ctx context.Context, q GetRolePermissionsQuery) ([]domain.Permission, error) {
	return scanPermissions(ctx, h.db,
		`SELECT p.id, p.name, p.description, p.created_at, p.updated_at
		   FROM permissions p
		   JOIN role_permissions rp ON rp.permission_id = p.id
		  WHERE rp.role_id = $1
		  ORDER BY p.name`, q.RoleID)
}

// AssignPermissionCommand binds a permission to a role.
type AssignPermissionCommand struct {
	RoleID       uuid.UUID
	PermissionID uuid.UUID
}

// AssignPermissionHandler binds a permission to a role.
type AssignPermissionHandler struct{ db postgres.Querier }

func NewAssignPermissionHandler(db postgres.Querier) AssignPermissionHandler {
	return AssignPermissionHandler{db: db}
}

// Handle binds the permission, tolerating a repeat bind.
func (h AssignPermissionHandler) Handle(ctx context.Context, cmd AssignPermissionCommand) error {
	_, err := h.db.Exec(ctx,
		`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		cmd.RoleID, cmd.PermissionID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return apperrors.New(apperrors.Invalid, "role or permission does not exist")
	}
	if err != nil {
		return apperrors.Wrap(apperrors.Internal, "assign permission", err)
	}
	return nil
}

// RemovePermissionCommand unbinds a permission from a role.
type RemovePermissionCommand struct {
	RoleID       uuid.UUID
	PermissionID uuid.UUID
}

// RemovePermissionHandler unbinds a permission from a role.
type RemovePermissionHandler struct{ db postgres.Querier }

func NewRemovePermissionHandler(db postgres.Querier) RemovePermissionHandler {
	return RemovePermissionHandler{db: db}
}

// Handle unbinds the permission, reporting NotFound when no binding existed.
func (h RemovePermissionHandler) Handle(ctx context.Context, cmd RemovePermissionCommand) error {
	tag, err := h.db.Exec(ctx,
		`DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`,
		cmd.RoleID, cmd.PermissionID)
	if err != nil {
		return apperrors.Wrap(apperrors.Internal, "remove permission", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.New(apperrors.NotFound, "role does not have this permission")
	}
	return nil
}
