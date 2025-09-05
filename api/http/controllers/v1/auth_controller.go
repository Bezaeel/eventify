package controllers

import (
	"eventify/internal/shared/auth"
	"eventify/internal/domain"
	"eventify/internal/service"
	"eventify/pkg"
	"eventify/pkg/logger"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Request and response types
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SignupRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

type AuthResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	UserID       uuid.UUID `json:"user_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type UserProfileResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type AuthController struct {
	router            fiber.Router
	UserService       *service.UserService
	jwtProvider       *auth.JWTProvider
	permissionService *service.PermissionService
	log               *logger.Logger
}

func NewAuthController(
	app *fiber.App,
	UserService *service.UserService,
	jwtProvider *auth.JWTProvider,
	permissionService *service.PermissionService,
	log *logger.Logger,
) *AuthController {
	return &AuthController{
		router:            app.Group("/api/v1/auth"),
		UserService:       UserService,
		jwtProvider:       jwtProvider,
		permissionService: permissionService,
		log:               log,
	}
}

func (ac *AuthController) RegisterRoutes() {
	ac.router.Post("/login", ac.Login)
	ac.router.Post("/signup", ac.Signup)
	ac.router.Post("/refresh", ac.RefreshToken)
	ac.router.Get("/me", ac.GetCurrentUser)
}

// Login godoc
// @Summary User login
// @Description Authenticate a user and return a JWT token
// @xtags v2
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} pkg.ErrorResponse
// @Failure 401 {object} pkg.ErrorResponse
// @Router /api/v1/auth/login [post]
func (ac *AuthController) Login(c *fiber.Ctx) error {
	var request LoginRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	ac.log.WithFields(logger.Fields{
		"email": request.Email,
	}).Info("User login attempt")

	// Validate the user credentials
	user, err := ac.UserService.CheckPassword(request.Email, request.Password)
	if err != nil {
		ac.log.WithFields(logger.Fields{
			"email": request.Email,
			"error": err.Error(),
		}).Error("first Login request body parsing error")

		ac.log.WithFields(logger.Fields{
			"email": request.Email,
		}).ErrorWithError("Login request body parsing error", err)
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "Invalid credentials",
		})
	}

	// Get user permissions
	permissions, err := ac.permissionService.GetPermissions(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error fetching user permissions",
		})
	}

	// Generate JWT token
	token, err := ac.jwtProvider.GenerateToken(user, permissions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error generating token",
		})
	}

	// For now, we'll use the same token as refresh token
	// In a production app, you'd create a separate refresh token system
	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour), // 1 hour expiry
	})
}

// Signup godoc
// @Summary User registration
// @Description Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param user body SignupRequest true "User information"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} pkg.ErrorResponse
// @Failure 409 {object} pkg.ErrorResponse
// @Router /api/v1/auth/signup [post]
func (ac *AuthController) Signup(c *fiber.Ctx) error {
	var request SignupRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	// Check if user already exists
	existingUser, _ := ac.UserService.GetByEmail(request.Email)
	if existingUser != nil {
		return c.Status(fiber.StatusConflict).JSON(pkg.ErrorResponse{
			Message: "User with this email already exists",
		})
	}

	// Create new user
	user := &domain.User{
		ID:        uuid.New(),
		Email:     request.Email,
		Password:  request.Password, // Will be hashed in service
		FirstName: request.FirstName,
		LastName:  request.LastName,
	}

	if err := ac.UserService.Create(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error creating user",
			Error:   err.Error(),
		})
	}

	// Generate JWT token
	token, err := ac.jwtProvider.GenerateToken(user, []string{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error generating token",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour), // 1 hour expiry
	})
}

// RefreshToken godoc
// @Summary Refresh authentication token
// @Description Get a new JWT token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh_token body RefreshRequest true "Refresh token"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} pkg.ErrorResponse
// @Failure 401 {object} pkg.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (ac *AuthController) RefreshToken(c *fiber.Ctx) error {
	var request RefreshRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	// Validate refresh token
	claims, err := ac.jwtProvider.ValidateToken(request.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "Invalid refresh token",
		})
	}

	// Get user from database
	user, err := ac.UserService.GetByID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "User not found",
		})
	}

	// Get user permissions
	permissions, err := ac.permissionService.GetPermissions(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error fetching user permissions",
		})
	}

	// Generate new token
	token, err := ac.jwtProvider.GenerateToken(user, permissions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error generating token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour), // 1 hour expiry
	})
}

// GetCurrentUser godoc
// @Summary Get current user profile
// @Description Get the profile of the currently authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserProfileResponse
// @Failure 401 {object} pkg.ErrorResponse
// @Router /api/v1/auth/me [get]
func (ac *AuthController) GetCurrentUser(c *fiber.Ctx) error {
	// The JWT middleware sets the claims in the context
	claims, ok := c.Locals("claims").(*auth.CustomClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "Unauthorized",
		})
	}

	// Get user from database
	user, err := ac.UserService.GetByID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "User not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
	})
}
