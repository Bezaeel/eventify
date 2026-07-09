package users

import (
	"context"
	"time"

	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SignupCommand registers a new account.
type SignupCommand struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// SignupResult identifies the created account.
type SignupResult struct {
	CreatedAt time.Time
	UserID    uuid.UUID
}

// SignupHandler creates a user.
type SignupHandler struct {
	db postgres.Querier
}

func NewSignupHandler(db postgres.Querier) SignupHandler {
	return SignupHandler{db: db}
}

// Handle hashes the password and inserts the user.
//
// A duplicate email is detected by the UNIQUE constraint, not by a preceding
// SELECT. The old AuthController.Signup called GetByEmail first and returned
// 409 if it found anything — but two concurrent requests both find nothing,
// both insert, and one gets a raw driver error rendered as a 500.
func (h SignupHandler) Handle(ctx context.Context, cmd SignupCommand) (SignupResult, error) {
	var res SignupResult

	if len(cmd.Password) < 8 {
		return res, apperrors.New(apperrors.Invalid, "password must be at least 8 characters")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "hash password", err)
	}

	err = h.db.QueryRow(ctx,
		`INSERT INTO users (id, email, password, first_name, last_name, created_at)
		 VALUES ($1, $2, $3, $4, $5, now())
		 RETURNING id, created_at`,
		uuid.New(), cmd.Email, string(hashed), cmd.FirstName, cmd.LastName,
	).Scan(&res.UserID, &res.CreatedAt)

	if isUniqueViolation(err) {
		return res, apperrors.New(apperrors.Conflict, "a user with this email already exists")
	}
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "insert user", err)
	}
	return res, nil
}
