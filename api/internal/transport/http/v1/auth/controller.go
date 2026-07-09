// Package auth is the HTTP v1 adapter for signup, login and profile.
package auth

import (
	"context"
	"time"

	"eventify/api/internal/domain"
	"eventify/api/internal/features/users"
	sharedauth "eventify/api/internal/shared/auth"
	"eventify/api/internal/transport/http/httperr"
	"eventify/api/internal/transport/http/middleware"
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handlers are the use cases this controller exposes, as function values so a
// test can inject a stub. See the Handlers doc in transport/http/v1/events.
type Handlers struct {
	Login   func(context.Context, users.LoginCommand) (users.LoginResult, error)
	Signup  func(context.Context, users.SignupCommand) (users.SignupResult, error)
	GetUser func(context.Context, users.GetUserQuery) (domain.User, error)
}

// Controller adapts HTTP onto the user use cases.
type Controller struct {
	h   Handlers
	jwt sharedauth.IJWTProvider
}

// New builds the controller.
func New(h Handlers, jwtProvider sharedauth.IJWTProvider) *Controller {
	return &Controller{h: h, jwt: jwtProvider}
}

// Register mounts the auth routes.
//
// /refresh is deliberately absent. The old endpoint accepted any valid access
// token as a "refresh token" — the comment said "For now, we'll use the same
// token as refresh token" — so a leaked access token could be renewed forever,
// and revocation was impossible. Refresh needs its own token type, stored and
// revocable server-side. Until that exists, the endpoint is worse than nothing.
func (c *Controller) Register(app *fiber.App) {
	r := app.Group("/api/v1/auth")

	r.Post("/login", c.Login)
	r.Post("/signup", c.Signup)
	r.Get("/me", middleware.JWT(c.jwt), c.Me)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	ExpiresAt time.Time `json:"expires_at"`
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
}

type signupRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type signupResponse struct {
	CreatedAt time.Time `json:"created_at"`
	UserID    uuid.UUID `json:"user_id"`
}

type profileResponse struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

// Login godoc
// @Summary User login
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body loginRequest true "Login credentials"
// @Success 200 {object} authResponse
// @Failure 401 {object} httperr.ErrorResponse
// @Router /api/v1/auth/login [post]
func (c *Controller) Login(ctx *fiber.Ctx) error {
	var req loginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid request body"))
	}

	res, err := c.h.Login(ctx.UserContext(), users.LoginCommand{
		Email: req.Email, Password: req.Password,
	})
	if err != nil {
		return httperr.Write(ctx, err)
	}

	// ExpiresAt comes from the provider, so it can no longer disagree with the
	// exp claim inside the token itself.
	return ctx.Status(fiber.StatusOK).JSON(authResponse{
		Token: res.Token, UserID: res.User.ID, ExpiresAt: res.ExpiresAt,
	})
}

// Signup godoc
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param user body signupRequest true "User information"
// @Success 201 {object} signupResponse
// @Failure 409 {object} httperr.ErrorResponse
// @Router /api/v1/auth/signup [post]
func (c *Controller) Signup(ctx *fiber.Ctx) error {
	var req signupRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid request body"))
	}

	res, err := c.h.Signup(ctx.UserContext(), users.SignupCommand{
		Email: req.Email, Password: req.Password,
		FirstName: req.FirstName, LastName: req.LastName,
	})
	if err != nil {
		return httperr.Write(ctx, err)
	}

	// No token is issued here. The old Signup minted one with an empty
	// permission set, which is indistinguishable from a user whose roles have
	// not loaded. Signing up and logging in are separate actions.
	return ctx.Status(fiber.StatusCreated).JSON(signupResponse{
		UserID: res.UserID, CreatedAt: res.CreatedAt,
	})
}

// Me godoc
// @Summary Get the current user's profile
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} profileResponse
// @Failure 401 {object} httperr.ErrorResponse
// @Router /api/v1/auth/me [get]
func (c *Controller) Me(ctx *fiber.Ctx) error {
	claims, ok := middleware.Claims(ctx)
	if !ok {
		return httperr.Write(ctx, apperrors.New(apperrors.Unauthorized, "authentication required"))
	}

	u, err := c.h.GetUser(ctx.UserContext(), users.GetUserQuery{UserID: claims.UserID})
	if err != nil {
		return httperr.Write(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(profileResponse{
		ID: u.ID, Email: u.Email, FirstName: u.FirstName, LastName: u.LastName, CreatedAt: u.CreatedAt,
	})
}
