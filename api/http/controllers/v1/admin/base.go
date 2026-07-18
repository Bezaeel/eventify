package controllers

import (
	"eventify/api/http/middlewares"
	"eventify/internal/shared/auth"
	"eventify/internal/shared/constants"
	"eventify/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AdminController struct {
	router            fiber.Router
	UserService       service.IUserService
	RoleService       service.IRoleService
	PermissionService service.IPermissionService
	jwtProvider       auth.IJWTProvider
}

func NewAdminController(
	app *fiber.App,
	userService service.IUserService,
	roleService service.IRoleService,
	permissionService service.IPermissionService,
	jwtProvider auth.IJWTProvider,
) *AdminController {
	return &AdminController{
		router:            app.Group("/api/v1/admin"),
		UserService:       userService,
		RoleService:       roleService,
		PermissionService: permissionService,
		jwtProvider:       jwtProvider,
	}
}

func (ac *AdminController) RegisterRoutes() {
	// Apply JWT middleware to all routes
	ac.router.Use(middlewares.JWTMiddleware(ac.jwtProvider))

	// All routes require admin permission
	adminMiddleware := middlewares.HasPermission(constants.Permissions.AdminPermission)

	// Register role routes
	ac.registerRoleRoutes(adminMiddleware)

	// Register permission routes
	ac.registerPermissionRoutes(adminMiddleware)

	// Register user routes
	ac.registerUserRoutes(adminMiddleware)
}
