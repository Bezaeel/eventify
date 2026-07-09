package auth_test

import (
	"testing"
	"time"

	"eventify/api/internal/domain"
	"eventify/api/internal/shared/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const secret = "a-test-secret-that-is-at-least-32-bytes-long"

func provider(t *testing.T) *auth.JWTProvider {
	t.Helper()
	p, err := auth.NewJWTProvider(secret, 60, "eventify", "eventify-api")
	require.NoError(t, err)
	return p
}

// The provider used to fall back to the literal
// "default-secret-key-change-in-production" when no secret was supplied, and
// cmd/http-server passed os.Getenv("JWT_SECRET") straight in. An unset variable
// therefore produced a server signing tokens with a constant published in this
// repository, letting anyone mint an admin token. It must fail closed.
func TestNewJWTProvider_FailsClosedWithoutAUsableSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{"empty secret", ""},
		{"secret shorter than 32 bytes", "too-short"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := auth.NewJWTProvider(tt.secret, 60, "eventify", "eventify-api")
			require.Error(t, err)
			require.Nil(t, p, "a provider that cannot sign securely must not exist")
		})
	}
}

func TestValidateToken_AcceptsATokenItIssued(t *testing.T) {
	p := provider(t)
	user := &domain.User{ID: uuid.New(), Email: "a@example.com"}

	token, err := p.GenerateToken(user, []string{"events.update"})
	require.NoError(t, err)

	claims, err := p.ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, user.ID, claims.UserID)
	require.Equal(t, []string{"events.update"}, claims.Permissions)
}

// ParseWithClaims hands the key callback whatever algorithm the token header
// declares. Without WithValidMethods, an attacker picks the algorithm.
func TestValidateToken_RejectsAlgNone(t *testing.T) {
	p := provider(t)

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, auth.CustomClaims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "eventify",
			Audience:  []string{"eventify-api"},
		},
	})
	raw, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = p.ValidateToken(raw)
	require.Error(t, err, "alg=none must never validate")
}

func TestValidateToken_RejectsAForeignSecret(t *testing.T) {
	issuer, err := auth.NewJWTProvider("another-secret-that-is-at-least-32-bytes!", 60, "eventify", "eventify-api")
	require.NoError(t, err)

	token, err := issuer.GenerateToken(&domain.User{ID: uuid.New()}, nil)
	require.NoError(t, err)

	_, err = provider(t).ValidateToken(token)
	require.Error(t, err)
}

// Issuer and audience were set at signing time and never verified, so a token
// minted by any other eventify-family service was accepted here.
func TestValidateToken_RejectsWrongIssuerAndAudience(t *testing.T) {
	tests := []struct {
		name     string
		issuer   string
		audience string
	}{
		{"wrong issuer", "someone-else", "eventify-api"},
		{"wrong audience", "eventify", "another-service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foreign, err := auth.NewJWTProvider(secret, 60, tt.issuer, tt.audience)
			require.NoError(t, err)

			token, err := foreign.GenerateToken(&domain.User{ID: uuid.New()}, nil)
			require.NoError(t, err)

			_, err = provider(t).ValidateToken(token)
			require.Error(t, err)
		})
	}
}

func TestValidateToken_RejectsAnExpiredToken(t *testing.T) {
	// -1 minute is coerced to the 60-minute default, so expiry is forced by
	// signing a claim set directly.
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.CustomClaims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			Issuer:    "eventify",
			Audience:  []string{"eventify-api"},
		},
	})
	raw, err := expired.SignedString([]byte(secret))
	require.NoError(t, err)

	_, err = provider(t).ValidateToken(raw)
	require.Error(t, err)
}

func TestExpiry_ReportsTheConfiguredDuration(t *testing.T) {
	p, err := auth.NewJWTProvider(secret, 15, "eventify", "eventify-api")
	require.NoError(t, err)
	require.Equal(t, 15*time.Minute, p.Expiry())
}
