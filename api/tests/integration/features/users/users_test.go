package users_test

import (
	"context"
	"testing"
	"time"

	"eventify/api/internal/features/users"
	"eventify/api/internal/shared/auth"
	"eventify/api/tests/integration/testsupport"
	"eventify/platform/apperrors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const testSecret = "a-test-secret-that-is-at-least-32-bytes-long"

func newJWT(t *testing.T) *auth.JWTProvider {
	t.Helper()
	p, err := auth.NewJWTProvider(testSecret, 60, "eventify", "eventify-api")
	require.NoError(t, err)
	return p
}

func TestIntegrationSignup(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	h := users.NewSignupHandler(pool)

	t.Run("creates a user and stores a bcrypt hash, never the plaintext", func(t *testing.T) {
		res, err := h.Handle(ctx, users.SignupCommand{
			Email: "a@example.com", Password: "correcthorse", FirstName: "A", LastName: "B",
		})
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, res.UserID)

		var stored string
		require.NoError(t, pool.QueryRow(ctx, `SELECT password FROM users WHERE id = $1`, res.UserID).Scan(&stored))
		require.NotEqual(t, "correcthorse", stored)
		require.Contains(t, stored, "$2a$", "must be a bcrypt hash")
	})

	t.Run("duplicate email is Conflict, resolved by the unique constraint", func(t *testing.T) {
		// The old Signup did SELECT-then-INSERT, which races: two concurrent
		// requests both see no row, both insert, one gets a raw driver error
		// rendered as a 500. The constraint decides here.
		_, err := h.Handle(ctx, users.SignupCommand{Email: "dup@example.com", Password: "correcthorse"})
		require.NoError(t, err)

		_, err = h.Handle(ctx, users.SignupCommand{Email: "dup@example.com", Password: "correcthorse"})
		require.Equal(t, apperrors.Conflict, apperrors.KindOf(err))
	})

	t.Run("rejects a short password", func(t *testing.T) {
		_, err := h.Handle(ctx, users.SignupCommand{Email: "s@example.com", Password: "short"})
		require.Equal(t, apperrors.Invalid, apperrors.KindOf(err))
	})
}

func TestIntegrationLogin(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	jwtProvider := newJWT(t)
	_, err := users.NewSignupHandler(pool).Handle(ctx, users.SignupCommand{
		Email: "login@example.com", Password: "correcthorse", FirstName: "L", LastName: "I",
	})
	require.NoError(t, err)

	h := users.NewLoginHandler(pool, jwtProvider)

	t.Run("issues a token whose expiry matches the provider", func(t *testing.T) {
		res, err := h.Handle(ctx, users.LoginCommand{Email: "login@example.com", Password: "correcthorse"})
		require.NoError(t, err)
		require.NotEmpty(t, res.Token)

		claims, err := jwtProvider.ValidateToken(res.Token)
		require.NoError(t, err)
		require.Equal(t, res.User.ID, claims.UserID)

		// The old AuthResponse hardcoded time.Now().Add(time.Hour) regardless of
		// the configured expiry, so the advertised expiry could disagree with
		// the exp claim inside the token.
		require.WithinDuration(t, claims.ExpiresAt.Time, res.ExpiresAt, 2*time.Second)
	})

	t.Run("a wrong password and an unknown email are indistinguishable", func(t *testing.T) {
		// Otherwise the endpoint is an account enumerator.
		_, wrongPass := h.Handle(ctx, users.LoginCommand{Email: "login@example.com", Password: "nope"})
		_, unknown := h.Handle(ctx, users.LoginCommand{Email: "ghost@example.com", Password: "correcthorse"})

		require.Equal(t, apperrors.Unauthorized, apperrors.KindOf(wrongPass))
		require.Equal(t, apperrors.Unauthorized, apperrors.KindOf(unknown))
		require.Equal(t, wrongPass.Error(), unknown.Error())
	})
}

func TestIntegrationChangePassword(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	created, err := users.NewSignupHandler(pool).Handle(ctx, users.SignupCommand{
		Email: "pw@example.com", Password: "correcthorse",
	})
	require.NoError(t, err)

	h := users.NewChangePasswordHandler(pool)
	login := users.NewLoginHandler(pool, newJWT(t))

	t.Run("rejects a wrong current password", func(t *testing.T) {
		err := h.Handle(ctx, users.ChangePasswordCommand{
			UserID: created.UserID, CurrentPassword: "wrong", NewPassword: "newpassword1",
		})
		require.Equal(t, apperrors.Unauthorized, apperrors.KindOf(err))
	})

	t.Run("rotates the password", func(t *testing.T) {
		require.NoError(t, h.Handle(ctx, users.ChangePasswordCommand{
			UserID: created.UserID, CurrentPassword: "correcthorse", NewPassword: "newpassword1",
		}))

		_, err := login.Handle(ctx, users.LoginCommand{Email: "pw@example.com", Password: "correcthorse"})
		require.Equal(t, apperrors.Unauthorized, apperrors.KindOf(err), "old password must stop working")

		_, err = login.Handle(ctx, users.LoginCommand{Email: "pw@example.com", Password: "newpassword1"})
		require.NoError(t, err)
	})

	t.Run("unknown user is NotFound", func(t *testing.T) {
		err := h.Handle(ctx, users.ChangePasswordCommand{
			UserID: uuid.New(), CurrentPassword: "x", NewPassword: "newpassword1",
		})
		require.Equal(t, apperrors.NotFound, apperrors.KindOf(err))
	})
}
