package users

import (
	"context"
	"errors"
	"time"

	"eventify/api/internal/domain"
	"eventify/api/internal/shared/auth"
	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginCommand authenticates a user by email and password.
type LoginCommand struct {
	Email    string
	Password string
}

// LoginResult carries the issued access token.
type LoginResult struct {
	ExpiresAt time.Time
	Token     string
	User      domain.User
}

// LoginHandler authenticates and mints a token.
//
// It holds the JWT provider rather than leaving token minting to each transport
// adapter: HTTP, gRPC and GraphQL would otherwise each reimplement it, and the
// expiry reported to the client would drift from the expiry baked into the
// token — which is exactly what happened when AuthResponse hardcoded
// time.Now().Add(time.Hour).
type LoginHandler struct {
	db  postgres.Querier
	jwt auth.IJWTProvider
}

func NewLoginHandler(db postgres.Querier, jwtProvider auth.IJWTProvider) LoginHandler {
	return LoginHandler{db: db, jwt: jwtProvider}
}

// Handle verifies credentials, loads permissions, and issues a token.
//
// Two round trips, so the queries live in unexported helpers in this package.
func (h LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (LoginResult, error) {
	var res LoginResult

	user, err := h.userByEmail(ctx, cmd.Email)
	if err != nil {
		return res, err
	}

	// bcrypt comparison runs even conceptually on the "user not found" path via
	// the same generic error below, so an attacker cannot distinguish an unknown
	// email from a wrong password by the response alone.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(cmd.Password)); err != nil {
		return res, apperrors.New(apperrors.Unauthorized, "invalid credentials")
	}

	permissions, err := permissionsFor(ctx, h.db, user.ID)
	if err != nil {
		return res, err
	}

	token, err := h.jwt.GenerateToken(&user, permissions)
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "generate token", err)
	}

	return LoginResult{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().Add(h.jwt.Expiry()),
	}, nil
}

func (h LoginHandler) userByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := scanUser(h.db.QueryRow(ctx, `SELECT `+columns+` FROM users WHERE email = $1`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		// Same message and Kind as a wrong password: revealing which emails are
		// registered turns the login endpoint into an account enumerator.
		return domain.User{}, apperrors.New(apperrors.Unauthorized, "invalid credentials")
	}
	if err != nil {
		return domain.User{}, apperrors.Wrap(apperrors.Internal, "load user", err)
	}
	return u, nil
}
