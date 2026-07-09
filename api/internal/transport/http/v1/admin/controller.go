// Package admin is the HTTP v1 adapter for role and permission administration.
package admin

import (
	"context"

	"eventify/api/internal/domain"
	"eventify/api/internal/features/permissions"
	"eventify/api/internal/features/roles"
	"eventify/api/internal/shared/auth"
	"eventify/api/internal/shared/constants"
	"eventify/api/internal/transport/http/httperr"
	"eventify/api/internal/transport/http/middleware"
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handlers are the use cases this controller exposes, as function values so a
// test can inject a stub. See the Handlers doc in transport/http/v1/events.
type Handlers struct {
	ListRoles        func(context.Context) ([]domain.Role, error)
	CreateRole       func(context.Context, roles.CreateRoleCommand) (roles.CreateRoleResult, error)
	AssignRole       func(context.Context, roles.AssignRoleCommand) error
	RemoveRole       func(context.Context, roles.RemoveRoleCommand) error
	GetUserRoles     func(context.Context, roles.GetUserRolesQuery) ([]domain.Role, error)
	ListPermissions  func(context.Context) ([]domain.Permission, error)
	RolePermissions  func(context.Context, permissions.GetRolePermissionsQuery) ([]domain.Permission, error)
	AssignPermission func(context.Context, permissions.AssignPermissionCommand) error
	RemovePermission func(context.Context, permissions.RemovePermissionCommand) error
}

// Controller adapts HTTP onto the admin use cases.
type Controller struct {
	h Handlers
}

// New builds the controller.
func New(h Handlers) *Controller { return &Controller{h: h} }

// Register mounts the admin routes. Every route requires authentication first,
// then an admin permission.
func (c *Controller) Register(app *fiber.App, jwtProvider auth.IJWTProvider) {
	r := app.Group("/api/v1/admin",
		middleware.JWT(jwtProvider),
		middleware.HasPermission(constants.Permissions.AdminPermission...),
	)

	r.Get("/roles", c.ListRoles)
	r.Post("/roles", c.CreateRole)
	r.Get("/roles/:id/permissions", c.RolePermissions)

	r.Get("/users/:id/roles", c.GetUserRoles)
	r.Post("/assign-role", c.AssignRole)
	r.Delete("/remove-role", c.RemoveRole)

	r.Get("/permissions", c.ListPermissions)
	r.Post("/assign-permission", c.AssignPermission)
	r.Delete("/remove-permission", c.RemovePermission)
}

// ---- DTOs ------------------------------------------------------------------

type roleResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type permissionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type createRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type assignRoleRequest struct {
	UserID uuid.UUID `json:"user_id"`
	RoleID uuid.UUID `json:"role_id"`
}

type assignPermissionRequest struct {
	RoleID       uuid.UUID `json:"role_id"`
	PermissionID uuid.UUID `json:"permission_id"`
}

func toRoles(in []domain.Role) []roleResponse {
	out := make([]roleResponse, 0, len(in))
	for _, r := range in {
		out = append(out, roleResponse{ID: r.ID, Name: r.Name, Description: r.Description})
	}
	return out
}

func toPermissions(in []domain.Permission) []permissionResponse {
	out := make([]permissionResponse, 0, len(in))
	for _, p := range in {
		out = append(out, permissionResponse{ID: p.ID, Name: p.Name, Description: p.Description})
	}
	return out
}

// ---- endpoints -------------------------------------------------------------

func (c *Controller) ListRoles(ctx *fiber.Ctx) error {
	rs, err := c.h.ListRoles(ctx.UserContext())
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(toRoles(rs))
}

func (c *Controller) CreateRole(ctx *fiber.Ctx) error {
	var req createRoleRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid request body"))
	}

	res, err := c.h.CreateRole(ctx.UserContext(), roles.CreateRoleCommand{
		Name: req.Name, Description: req.Description,
	})
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"role_id": res.RoleID, "created_at": res.CreatedAt})
}

func (c *Controller) RolePermissions(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid role id"))
	}

	ps, err := c.h.RolePermissions(ctx.UserContext(), permissions.GetRolePermissionsQuery{RoleID: id})
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(toPermissions(ps))
}

func (c *Controller) GetUserRoles(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid user id"))
	}

	rs, err := c.h.GetUserRoles(ctx.UserContext(), roles.GetUserRolesQuery{UserID: id})
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(toRoles(rs))
}

func (c *Controller) AssignRole(ctx *fiber.Ctx) error {
	var req assignRoleRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid request body"))
	}

	if err := c.h.AssignRole(ctx.UserContext(), roles.AssignRoleCommand{
		UserID: req.UserID, RoleID: req.RoleID,
	}); err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *Controller) RemoveRole(ctx *fiber.Ctx) error {
	var req assignRoleRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid request body"))
	}

	if err := c.h.RemoveRole(ctx.UserContext(), roles.RemoveRoleCommand{
		UserID: req.UserID, RoleID: req.RoleID,
	}); err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *Controller) ListPermissions(ctx *fiber.Ctx) error {
	ps, err := c.h.ListPermissions(ctx.UserContext())
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(toPermissions(ps))
}

func (c *Controller) AssignPermission(ctx *fiber.Ctx) error {
	var req assignPermissionRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid request body"))
	}

	if err := c.h.AssignPermission(ctx.UserContext(), permissions.AssignPermissionCommand{
		RoleID: req.RoleID, PermissionID: req.PermissionID,
	}); err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *Controller) RemovePermission(ctx *fiber.Ctx) error {
	var req assignPermissionRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid request body"))
	}

	if err := c.h.RemovePermission(ctx.UserContext(), permissions.RemovePermissionCommand{
		RoleID: req.RoleID, PermissionID: req.PermissionID,
	}); err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
