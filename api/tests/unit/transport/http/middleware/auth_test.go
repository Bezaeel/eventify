package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"eventify/api/internal/domain"
	"eventify/api/internal/shared/auth"
	"eventify/api/internal/transport/http/middleware"

	"github.com/gofiber/fiber/v2"
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

func tokenWith(t *testing.T, p *auth.JWTProvider, perms ...string) string {
	t.Helper()
	tok, err := p.GenerateToken(&domain.User{ID: uuid.New(), Email: "a@example.com"}, perms)
	require.NoError(t, err)
	return tok
}

func ok(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) }

func TestJWT_RejectsMissingOrMalformedHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"absent", ""},
		{"no bearer prefix", "abc.def.ghi"},
		{"wrong scheme", "Basic abc"},
		{"bearer with no token", "Bearer "},
	}

	p := provider(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/x", middleware.JWT(p), ok)

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestJWT_AcceptsAValidTokenAndPopulatesClaims(t *testing.T) {
	p := provider(t)
	app := fiber.New()
	app.Get("/x", middleware.JWT(p), func(c *fiber.Ctx) error {
		claims, found := middleware.Claims(c)
		require.True(t, found)
		require.Equal(t, "a@example.com", claims.Email)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tokenWith(t, p))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHasPermission(t *testing.T) {
	p := provider(t)

	tests := []struct {
		name    string
		granted []string
		require string
		want    int
	}{
		{"granted", []string{"events.update"}, "events.update", http.StatusOK},
		{"not granted", []string{"events.read"}, "events.update", http.StatusForbidden},
		{"no permissions at all", nil, "events.update", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/x", middleware.JWT(p), middleware.HasPermission(tt.require), ok)

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.Header.Set("Authorization", "Bearer "+tokenWith(t, p, tt.granted...))
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			require.Equal(t, tt.want, resp.StatusCode)
		})
	}
}

// The v2 update route mounted HasPermission without JWT, so claims were never
// populated and the route answered 401 for everyone — it was unreachable for as
// long as it existed. Failing closed at request time hides the bug; the route
// is now rejected when it is built.
func TestHasPermission_WithoutJWT_FailsClosed(t *testing.T) {
	app := fiber.New()
	app.Get("/x", middleware.HasPermission("events.update"), ok)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/x", nil), -1)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"a route with permissions but no authentication must never allow the request through")
}

func TestHasPermission_PanicsWhenGivenNoPermissions(t *testing.T) {
	require.Panics(t, func() { middleware.HasPermission() },
		"an empty permission set would authorise everyone")
}
