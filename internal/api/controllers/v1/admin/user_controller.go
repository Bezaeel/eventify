package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// @Summary Register user routes
// @Description Registers all user-related routes with their respective middlewares
func (ac *AdminController) registerUserRoutes(adminMiddleware fiber.Handler) {
	ac.router.Post("/assign-role", adminMiddleware, ac.AssignRoleToUser)
	ac.router.Delete("/remove-role", adminMiddleware, ac.RemoveRoleFromUser)
	ac.router.Get("/users/:id/roles", adminMiddleware, ac.GetUserRoles)
}

// AssignRoleToUser godoc
// @Summary Assign role to user
// @Description Assigns a specific role to a user
// @Tags Admin-Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AssignRoleRequest true "Role assignment request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse "Invalid request or user/role not found"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/assign-role [post]
func (ac *AdminController) AssignRoleToUser(c *fiber.Ctx) error {
	request := new(AssignRoleRequest)
	if err := c.BodyParser(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	userID, err := uuid.Parse(request.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid user ID",
		})
	}

	roleID, err := uuid.Parse(request.RoleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid role ID",
		})
	}

	// Verify that both user and role exist
	_, err = ac.UserService.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "User not found",
		})
	}

	_, err = ac.RoleService.GetByID(roleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Role not found",
		})
	}

	if err := ac.RoleService.AssignRoleToUser(userID, roleID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Error assigning role to user",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Role assigned successfully",
	})
}

// RemoveRoleFromUser godoc
// @Summary Remove role from user
// @Description Removes a specific role from a user
// @Tags Admin-Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AssignRoleRequest true "Role removal request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/remove-role [delete]
func (ac *AdminController) RemoveRoleFromUser(c *fiber.Ctx) error {
	request := new(AssignRoleRequest)
	if err := c.BodyParser(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	userID, err := uuid.Parse(request.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid user ID",
		})
	}

	roleID, err := uuid.Parse(request.RoleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid role ID",
		})
	}

	if err := ac.RoleService.RemoveRoleFromUser(userID, roleID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Error removing role from user",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Role removed successfully",
	})
}

// GetUserRoles godoc
// @Summary Get user roles
// @Description Retrieves all roles assigned to a specific user
// @Tags Admin-Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID" format(uuid)
// @Success 200 {array} domain.Role
// @Failure 400 {object} ErrorResponse "Invalid user ID or user not found"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/users/{id}/roles [get]
func (ac *AdminController) GetUserRoles(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid user ID",
		})
	}

	// Verify that user exists
	_, err = ac.UserService.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "User not found",
		})
	}

	roles, err := ac.RoleService.GetUserRoles(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Error fetching user roles",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(roles)
}
