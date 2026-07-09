// Package users holds the user and authentication use cases.
//
// One file per use case, raw SQL inline. Where a use case needs more than one
// database round trip — login must fetch the user and then its permissions —
// the extra queries live in unexported helpers in the same file, not in a
// repository type.
package users

import (
	"context"
	"errors"

	"eventify/api/internal/domain"
	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const columns = `id, email, password, first_name, last_name, created_at, updated_at`

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

// permissionsFor returns the permission names granted to a user through its
// roles. Used by login and by token refresh.
func permissionsFor(ctx context.Context, db postgres.Querier, userID any) ([]string, error) {
	rows, err := db.Query(ctx,
		`SELECT p.name
		   FROM permissions p
		   JOIN role_permissions rp ON rp.permission_id = p.id
		   JOIN user_roles ur      ON ur.role_id = rp.role_id
		  WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.Internal, "load permissions", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, apperrors.Wrap(apperrors.Internal, "scan permission", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// uniqueViolation is the Postgres SQLSTATE for a unique-constraint breach.
const uniqueViolation = "23505"

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
//
// Signup previously did SELECT-then-INSERT to detect a duplicate email, which
// races: two concurrent signups both see no row and both insert. The users table
// has a UNIQUE constraint on email, so the correct move is to attempt the insert
// and translate the violation into a Conflict.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}
