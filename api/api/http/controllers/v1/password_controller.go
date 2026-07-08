package controllers

import (
	"eventify/api/http/middlewares"
	"eventify/internal/service"
	"eventify/internal/shared/auth"
	"eventify/pkg"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}

type PasswordController struct {
	router      fiber.Router
	UserService service.IUserService
	jwtProvider auth.IJWTProvider
}

func NewPasswordController(
	app *fiber.App,
	UserService service.IUserService,
	jwtProvider auth.IJWTProvider,
) *PasswordController {
	return &PasswordController{
		router:      app.Group("/api/v1/password"),
		UserService: UserService,
		jwtProvider: jwtProvider,
	}
}

func (pc *PasswordController) RegisterRoutes() {
	pc.router.Post("/forgot", pc.ForgotPassword)
	pc.router.Post("/reset", pc.ResetPassword)

	// Protected route - requires authentication
	pc.router.Post("/change", middlewares.JWTMiddleware(pc.jwtProvider), pc.ChangePassword)
}

// SuccessResponse represents a successful operation response
type SuccessResponse struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send a password reset link to user's email
// @Tags password
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Email for password reset"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} pkg.ErrorResponse
// @Failure 404 {object} pkg.ErrorResponse
// @Router /api/v1/password/forgot [post]
func (pc *PasswordController) ForgotPassword(c *fiber.Ctx) error {
	var request ForgotPasswordRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request",
		})
	}

	// Find user by email
	user, err := pc.UserService.GetByEmail(request.Email)
	if err != nil {
		// For security reasons, don't reveal if the email exists or not
		return c.Status(fiber.StatusOK).JSON(SuccessResponse{
			Message: "If the email exists, a password reset link will be sent",
		})
	}

	// Generate a password reset token
	resetToken, err := pc.jwtProvider.GenerateToken(user, []string{"password.reset"})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error generating reset token",
		})
	}

	// In a real app, you would send this via email
	// For now, just return it in the response (for testing)
	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Password reset link generated",
		Token:   resetToken, // In production, remove this line and send via email
	})
}

// ResetPassword godoc
// @Summary Reset password
// @Description Reset a user's password using a reset token
// @Tags password
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset token and new password"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} pkg.ErrorResponse
// @Failure 401 {object} pkg.ErrorResponse
// @Router /api/v1/password/reset [post]
func (pc *PasswordController) ResetPassword(c *fiber.Ctx) error {
	var request ResetPasswordRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request",
		})
	}

	// Validate token
	claims, err := pc.jwtProvider.ValidateToken(request.Token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "Invalid or expired token",
		})
	}

	// Check if token has the required permission
	hasPermission := false
	for _, p := range claims.Permissions {
		if p == "password.reset" {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "Invalid token type",
		})
	}

	// Update password
	err = pc.UserService.UpdatePassword(claims.UserID, request.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error updating password",
		})
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Password reset successfully",
	})
}

// ChangePassword godoc
// @Summary Change password
// @Description Change the user's password (requires authentication)
// @Tags password
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Current and new password"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} pkg.ErrorResponse
// @Failure 401 {object} pkg.ErrorResponse
// @Router /api/v1/password/change [post]
func (pc *PasswordController) ChangePassword(c *fiber.Ctx) error {
	var request ChangePasswordRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request",
		})
	}

	// Get user from JWT claims
	claims, ok := c.Locals("claims").(*auth.CustomClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "Authentication required",
		})
	}

	// Get user from database
	user, err := pc.UserService.GetByID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "User not found",
		})
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.CurrentPassword))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(pkg.ErrorResponse{
			Message: "Current password is incorrect",
		})
	}

	// Update password
	err = pc.UserService.UpdatePassword(user.ID, request.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error updating password",
		})
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Password changed successfully",
	})
}
