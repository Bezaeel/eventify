// Package password is the HTTP v1 adapter for password rotation.
package password

import (
	"context"

	"eventify/api/internal/features/users"
	sharedauth "eventify/api/internal/shared/auth"
	"eventify/api/internal/transport/http/httperr"
	"eventify/api/internal/transport/http/middleware"
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
)

// Handlers are the use cases this controller exposes, as function values so a
// test can inject a stub. See the Handlers doc in transport/http/v1/events.
type Handlers struct {
	ChangePassword func(context.Context, users.ChangePasswordCommand) error
}

// Controller adapts HTTP onto password rotation.
type Controller struct {
	h   Handlers
	jwt sharedauth.IJWTProvider
}

// New builds the controller.
func New(h Handlers, jwtProvider sharedauth.IJWTProvider) *Controller {
	return &Controller{h: h, jwt: jwtProvider}
}

// Register mounts the password routes.
//
// /forgot and /reset are deliberately absent.
//
// The previous /forgot handler generated a password-reset JWT and returned it
// **in the HTTP response body**, with the comment "In production, remove this
// line and send via email". Anyone who knew an email address could POST it and
// receive a token that reset that account's password — unauthenticated account
// takeover, reachable from the public internet.
//
// The token was also a full access token carrying a "password.reset"
// permission, so it authenticated against every endpoint that does not check
// permissions, including /api/v1/auth/me.
//
// Restoring this flow needs a single-use, short-lived, hashed reset token in
// its own table, delivered out of band. Until then there is no endpoint, which
// is strictly safer than an exploitable one.
func (c *Controller) Register(app *fiber.App) {
	r := app.Group("/api/v1/password")
	r.Post("/change", middleware.JWT(c.jwt), c.Change)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Change godoc
// @Summary Change the current user's password
// @Tags password
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body changePasswordRequest true "Current and new password"
// @Success 204
// @Failure 401 {object} httperr.ErrorResponse
// @Router /api/v1/password/change [post]
func (c *Controller) Change(ctx *fiber.Ctx) error {
	claims, ok := middleware.Claims(ctx)
	if !ok {
		return httperr.Write(ctx, apperrors.New(apperrors.Unauthorized, "authentication required"))
	}

	var req changePasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid request body"))
	}

	err := c.h.ChangePassword(ctx.UserContext(), users.ChangePasswordCommand{
		UserID:          claims.UserID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
