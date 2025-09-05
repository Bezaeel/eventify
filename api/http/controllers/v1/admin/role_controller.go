package controllers

import (
	"eventify/internal/domain"
	"eventify/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// @Summary Register role routes
// @Description Registers all role-related routes with their respective middlewares
func (ac *AdminController) registerRoleRoutes(adminMiddleware fiber.Handler) {
	ac.router.Get("/roles", adminMiddleware, ac.GetAllRoles)
	ac.router.Post("/roles", adminMiddleware, ac.CreateRole)
	ac.router.Get("/roles/:id/permissions", adminMiddleware, ac.GetRolePermissions)
}

// GetAllRoles godoc
// @Summary Get all roles
// @Description Retrieves a list of all roles in the system
// @Tags Admin-Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.Role
// @Failure 401 {object} pkg.ErrorResponse "Unauthorized"
// @Failure 403 {object} pkg.ErrorResponse "Forbidden"
// @Failure 500 {object} pkg.ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/roles [get]
func (ac *AdminController) GetAllRoles(c *fiber.Ctx) error {
	roles, err := ac.RoleService.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error fetching roles",
			Error:   err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(roles)
}

// CreateRole godoc
// @Summary Create a new role
// @Description Creates a new role in the system
// @Tags Admin-Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param role body CreateRoleRequest true "Role creation request"
// @Success 201 {object} domain.Role
// @Failure 400 {object} pkg.ErrorResponse "Invalid request"
// @Failure 401 {object} pkg.ErrorResponse "Unauthorized"
// @Failure 403 {object} pkg.ErrorResponse "Forbidden"
// @Failure 500 {object} pkg.ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/roles [post]
func (ac *AdminController) CreateRole(c *fiber.Ctx) error {
	request := new(CreateRoleRequest)
	if err := c.BodyParser(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	role := &domain.Role{
		Name:        request.Name,
		Description: request.Description,
	}

	if err := ac.RoleService.Create(role); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error creating role",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(role)
}

// GetRolePermissions godoc
// @Summary Get role permissions
// @Description Retrieves all permissions assigned to a specific role
// @Tags Admin-Roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID" format(uuid)
// @Success 200 {array} domain.Permission
// @Failure 400 {object} pkg.ErrorResponse "Invalid role ID or role not found"
// @Failure 401 {object} pkg.ErrorResponse "Unauthorized"
// @Failure 403 {object} pkg.ErrorResponse "Forbidden"
// @Failure 500 {object} pkg.ErrorResponse "Internal Server Error"
// @Router /api/v1/admin/roles/{id}/permissions [get]
func (ac *AdminController) GetRolePermissions(c *fiber.Ctx) error {
	id := c.Params("id")
	roleID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid role ID",
		})
	}

	// Verify that role exists
	_, err = ac.RoleService.GetByID(roleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Role not found",
		})
	}

	permissions, err := ac.PermissionService.GetRolePermissions(roleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: "Error fetching role permissions",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(permissions)
}
