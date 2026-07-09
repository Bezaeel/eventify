package users

import (
	"context"
	"errors"

	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ChangePasswordCommand rotates a password, proving knowledge of the old one.
type ChangePasswordCommand struct {
	CurrentPassword string
	NewPassword     string
	UserID          uuid.UUID
}

// ChangePasswordHandler verifies then rotates.
type ChangePasswordHandler struct {
	db postgres.Querier
}

func NewChangePasswordHandler(db postgres.Querier) ChangePasswordHandler {
	return ChangePasswordHandler{db: db}
}

// Handle checks the current password, then writes the new hash.
//
// Two round trips (read the hash, write the new one) so the read lives in an
// unexported helper. It is not wrapped in a transaction: the read is only a
// guard, and a concurrent rotation that wins the race simply means one of the
// two new passwords survives, which is the same outcome as serialising them.
func (h ChangePasswordHandler) Handle(ctx context.Context, cmd ChangePasswordCommand) error {
	if len(cmd.NewPassword) < 8 {
		return apperrors.New(apperrors.Invalid, "password must be at least 8 characters")
	}

	current, err := h.passwordHash(ctx, cmd.UserID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(current), []byte(cmd.CurrentPassword)); err != nil {
		return apperrors.New(apperrors.Unauthorized, "current password is incorrect")
	}

	return SetPassword(ctx, h.db, cmd.UserID, cmd.NewPassword)
}

func (h ChangePasswordHandler) passwordHash(ctx context.Context, id uuid.UUID) (string, error) {
	var hash string
	err := h.db.QueryRow(ctx, `SELECT password FROM users WHERE id = $1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperrors.New(apperrors.NotFound, "user not found")
	}
	if err != nil {
		return "", apperrors.Wrap(apperrors.Internal, "load password", err)
	}
	return hash, nil
}

// SetPassword hashes and stores a new password. Exported because the password
// reset flow, which authenticates by token rather than by old password, needs
// exactly this and nothing else.
func SetPassword(ctx context.Context, db postgres.Querier, id uuid.UUID, plaintext string) error {
	if len(plaintext) < 8 {
		return apperrors.New(apperrors.Invalid, "password must be at least 8 characters")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.Wrap(apperrors.Internal, "hash password", err)
	}

	tag, err := db.Exec(ctx,
		`UPDATE users SET password = $2, updated_at = now() WHERE id = $1`, id, string(hashed))
	if err != nil {
		return apperrors.Wrap(apperrors.Internal, "update password", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.New(apperrors.NotFound, "user not found")
	}
	return nil
}
