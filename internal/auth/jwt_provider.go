package auth

import (
	"errors"
	"eventify/internal/domain"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CustomClaims extends jwt.RegisteredClaims with user-specific claims
type CustomClaims struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Permissions []string  `json:"permissions"`
	jwt.RegisteredClaims
}

// JWTProvider handles JWT token generation and validation
type JWTProvider struct {
	secretKey []byte
	expiryMin int
	issuer    string
	audience  string
}

type IJWTProvider interface {
	GenerateToken(user *domain.User, permissions []string) (string, error)
	ValidateToken(tokenString string) (*CustomClaims, error)
}

// NewJWTProvider creates a new JWTProvider
func NewJWTProvider(secretKey string, expiryMin int, issuer, audience string) *JWTProvider {
	if secretKey == "" {
		secretKey = os.Getenv("JWT_SECRET")
		if secretKey == "" {
			secretKey = "default-secret-key-change-in-production"
		}
	}

	if expiryMin == 0 {
		expiryMin = 60 // Default to 60 minutes
	}

	return &JWTProvider{
		secretKey: []byte(secretKey),
		expiryMin: expiryMin,
		issuer:    issuer,
		audience:  audience,
	}
}

// GenerateToken creates a new JWT token for a user with permissions
func (j *JWTProvider) GenerateToken(user *domain.User, permissions []string) (string, error) {
	now := time.Now()
	claims := CustomClaims{
		UserID:      user.ID,
		Email:       user.Email,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.expiryMin) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    j.issuer,
			Audience:  []string{j.audience},
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateToken validates an incoming token and returns the claims
func (j *JWTProvider) ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
