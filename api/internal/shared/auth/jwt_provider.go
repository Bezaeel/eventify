// Package auth issues and validates JWT access tokens.
package auth

import (
	"errors"
	"fmt"
	"time"

	"eventify/api/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CustomClaims extends jwt.RegisteredClaims with user-specific claims.
type CustomClaims struct {
	Email       string    `json:"email"`
	Permissions []string  `json:"permissions"`
	UserID      uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTProvider issues and validates access tokens.
type JWTProvider struct {
	issuer    string
	audience  string
	secretKey []byte
	expiry    time.Duration
}

// IJWTProvider is the seam the HTTP middleware depends on.
type IJWTProvider interface {
	GenerateToken(user *domain.User, permissions []string) (string, error)
	ValidateToken(tokenString string) (*CustomClaims, error)
	Expiry() time.Duration
}

// NewJWTProvider builds a provider, refusing to start without a secret.
//
// The previous implementation fell back to os.Getenv("JWT_SECRET") and then, if
// that was also empty, to the literal "default-secret-key-change-in-production".
// Since cmd/http-server passed os.Getenv("JWT_SECRET") straight in, an unset
// variable silently produced a server signing tokens with a constant published
// in the source tree — anyone could mint an admin token. Failing closed is the
// only safe behaviour: a service that cannot sign securely must not serve.
func NewJWTProvider(secretKey string, expiryMins int, issuer, audience string) (*JWTProvider, error) {
	if secretKey == "" {
		return nil, errors.New("jwt secret must not be empty")
	}
	if len(secretKey) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 bytes, got %d", len(secretKey))
	}
	if expiryMins <= 0 {
		expiryMins = 60
	}

	return &JWTProvider{
		secretKey: []byte(secretKey),
		expiry:    time.Duration(expiryMins) * time.Minute,
		issuer:    issuer,
		audience:  audience,
	}, nil
}

// Expiry is how long an issued token remains valid.
//
// Transports read this rather than hardcoding a duration: the old AuthResponse
// reported `ExpiresAt: time.Now().Add(time.Hour)` no matter what expiry the
// provider was configured with.
func (j *JWTProvider) Expiry() time.Duration { return j.expiry }

// GenerateToken issues an access token for a user.
func (j *JWTProvider) GenerateToken(user *domain.User, permissions []string) (string, error) {
	now := time.Now()
	claims := CustomClaims{
		UserID:      user.ID,
		Email:       user.Email,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    j.issuer,
			Audience:  []string{j.audience},
			Subject:   user.ID.String(),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secretKey)
}

// ValidateToken parses and verifies a token.
//
// The signing method is pinned with WithValidMethods. Without it, ParseWithClaims
// hands the key callback whatever algorithm the token header claims, which is the
// classic JWT algorithm-confusion attack surface. Issuer and audience are checked
// too; previously they were set at signing time and never verified.
func (j *JWTProvider) ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(*jwt.Token) (any, error) { return j.secretKey, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(j.issuer),
		jwt.WithAudience(j.audience),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
