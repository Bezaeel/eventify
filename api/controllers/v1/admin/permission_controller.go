package controllers

import (
	"eventify/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// @Summary Register permission routes
// @Description Registers all permission-related routes with their respective middlewares
func (ac *AdminController) registerPermissionRoutes(adminMiddleware fiber.Handler) {
	ac.router.Get("/permissions", adminMiddleware, ac.GetAllPermissions)
	ac.router.Post("/assign-permission", adminMiddleware, ac.AssignPermissionToRole)
	ac.router.Delete("/remove-permission", adminMiddleware, ac.RemovePermissionFromRole)
}

// GetAllPermissions godoc
// @Summary Get all permissions
// @Description Retrieves a list of all permissions in the system
// @Tags Admin-Permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.Permission
// @Failure 401 {object} pkg.ErrorResponse "Unauthorized"
// @Failure 403 {object} pkg.ErrorResponse "Forbidden"
// @Failure 500 {object} pkg.ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/permissions [get]
func (ac *AdminController) GetAllPermissions(c *fiber.Ctx) error {
	permissions, err := ac.PermissionService.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error fetching permissions",
			Error:   err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(permissions)
}

// AssignPermissionToRole godoc
// @Summary Assign permission to role
// @Description Assigns a specific permission to a role
// @Tags Admin-Permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AssignPermissionRequest true "Permission assignment request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} pkg.ErrorResponse "Invalid request"
// @Failure 401 {object} pkg.ErrorResponse "Unauthorized"
// @Failure 403 {object} pkg.ErrorResponse "Forbidden"
// @Failure 500 {object} pkg.ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/assign-permission [post]
func (ac *AdminController) AssignPermissionToRole(c *fiber.Ctx) error {
	request := new(AssignPermissionRequest)
	if err := c.BodyParser(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	roleID, err := uuid.Parse(request.RoleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid role ID",
		})
	}

	permissionID, err := uuid.Parse(request.PermissionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid permission ID",
		})
	}

	if err := ac.PermissionService.AssignPermissionToRole(roleID, permissionID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error assigning permission to role",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Permission assigned successfully",
	})
}

// RemovePermissionFromRole godoc
// @Summary Remove permission from role
// @Description Removes a specific permission from a role
// @Tags Admin-Permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AssignPermissionRequest true "Permission removal request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} pkg.ErrorResponse "Invalid request"
// @Failure 401 {object} pkg.ErrorResponse "Unauthorized"
// @Failure 403 {object} pkg.ErrorResponse "Forbidden"
// @Failure 500 {object} pkg.ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/remove-permission [delete]
func (ac *AdminController) RemovePermissionFromRole(c *fiber.Ctx) error {
	request := new(AssignPermissionRequest)
	if err := c.BodyParser(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	roleID, err := uuid.Parse(request.RoleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid role ID",
		})
	}

	permissionID, err := uuid.Parse(request.PermissionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid permission ID",
		})
	}

	if err := ac.PermissionService.RemovePermissionFromRole(roleID, permissionID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error removing permission from role",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Message: "Permission removed successfully",
	})
}
