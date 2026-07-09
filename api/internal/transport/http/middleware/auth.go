// Package middleware holds the Fiber middleware shared by every HTTP version.
package middleware

import (
	"eventify/api/internal/shared/auth"
	"eventify/api/internal/transport/http/httperr"
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
)

// claimsKey is the Locals key under which validated claims are stored.
const claimsKey = "claims"

// JWT validates the bearer token and stores the claims for later middleware.
func JWT(jwtProvider auth.IJWTProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := bearerToken(c)
		if !ok {
			return httperr.Write(c, apperrors.New(apperrors.Unauthorized, "bearer token required"))
		}

		claims, err := jwtProvider.ValidateToken(token)
		if err != nil {
			return httperr.Write(c, apperrors.New(apperrors.Unauthorized, "invalid or expired token"))
		}

		c.Locals(claimsKey, claims)
		return c.Next()
	}
}

// bearerToken extracts the token from an `Authorization: Bearer <token>` header.
//
// The old implementation split on " " and required exactly two parts, so a
// header with a trailing space produced three parts and a 401 that looked like
// a bad token.
func bearerToken(c *fiber.Ctx) (string, bool) {
	const prefix = "Bearer "
	header := c.Get(fiber.HeaderAuthorization)
	if len(header) <= len(prefix) || !equalFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return header[len(prefix):], true
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// Claims returns the validated claims, if JWT ran on this route.
func Claims(c *fiber.Ctx) (*auth.CustomClaims, bool) {
	claims, ok := c.Locals(claimsKey).(*auth.CustomClaims)
	return claims, ok
}

// HasPermission rejects a request whose claims carry none of the permissions.
//
// It requires JWT to have run first. It used to be possible to register
// HasPermission without JWT — the v2 update-event route did exactly that — in
// which case Locals("claims") is nil and every request 401s. That route was
// unreachable for as long as it existed. Registering permissions without
// authentication is now a programming error, so this panics at route-build time
// rather than failing silently at request time.
func HasPermission(permissions ...string) fiber.Handler {
	if len(permissions) == 0 {
		panic("middleware.HasPermission: at least one permission required")
	}

	return func(c *fiber.Ctx) error {
		claims, ok := Claims(c)
		if !ok {
			// JWT did not run on this route. Fail closed, and make it loud in
			// the logs rather than looking like an ordinary auth failure.
			return httperr.Write(c, apperrors.New(apperrors.Unauthorized,
				"route misconfigured: HasPermission requires JWT middleware"))
		}

		for _, granted := range claims.Permissions {
			for _, required := range permissions {
				if granted == required {
					return c.Next()
				}
			}
		}
		return httperr.Write(c, apperrors.New(apperrors.Forbidden, "insufficient permissions"))
	}
}
