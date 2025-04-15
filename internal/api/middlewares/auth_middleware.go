package middlewares

import (
	"eventify/internal/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// JWTMiddleware creates middleware to validate JWT tokens
func JWTMiddleware(jwtProvider auth.IJWTProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Authorization header is required",
			})
		}

		// Check if the token is a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Bearer token format required",
			})
		}

		// Validate the token
		claims, err := jwtProvider.ValidateToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid or expired token",
			})
		}

		// Store the claims in context for later use
		c.Locals("claims", claims)
		return c.Next()
	}
}

// HasPermission creates middleware to check if a user has the required permission
func HasPermission(permission []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get claims from context
		claims, ok := c.Locals("claims").(*auth.CustomClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized: missing authentication",
			})
		}

		// Check if the user has the required permission
		for _, p := range claims.Permissions {
			for _, perm := range permission {
				if p == perm {
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Forbidden: insufficient permissions",
		})
	}
}
