package helpers

import (
	"eventify/internal/auth"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateValidToken(jwtProvider auth.IJWTProvider, permissions []string) (string, error) {
	claims := jwt.MapClaims{
		"sub":         uuid.New().String(),
		"exp":         time.Now().Add(time.Hour).Unix(),
		"iat":         time.Now().Unix(),
		"iss":         "test-issuer",
		"aud":         "test-audience",
		"permissions": permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("test-secret-key"))
}
