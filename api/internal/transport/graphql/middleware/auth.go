// Package middleware holds net/http middleware for the GraphQL server.
//
// The GraphQL server previously had no authentication of any kind. The
// createEvent resolver compensated by setting CreatedBy to uuid.New() behind a
// `TODO: Get from context`, attributing every event to a user that never
// existed.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"eventify/api/internal/shared/auth"
)

type ctxKey int

const claimsKey ctxKey = iota

// Claims returns the validated claims attached by Auth, if a valid token was
// presented.
func Claims(ctx context.Context) (*auth.CustomClaims, bool) {
	c, ok := ctx.Value(claimsKey).(*auth.CustomClaims)
	return c, ok
}

// Auth validates an optional bearer token and attaches its claims.
//
// It does not reject unauthenticated requests: GraphQL exposes public queries
// (`event`, `events`) and authenticated mutations through a single endpoint, so
// authorisation is a per-resolver decision. A missing or invalid token simply
// means no claims in context, and the mutations that need them fail with
// UNAUTHENTICATED.
func Auth(jwtProvider auth.IJWTProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, found := strings.CutPrefix(header, "Bearer ")
			if !found || token == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := jwtProvider.ValidateToken(token)
			if err != nil {
				// An invalid token is treated as no token. The resolver decides.
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
		})
	}
}
