// Package http wires the Fiber application: middleware, then every versioned
// controller.
package http

import (
	"eventify/api/internal/features/events"
	"eventify/api/internal/features/permissions"
	"eventify/api/internal/features/roles"
	"eventify/api/internal/features/users"
	"eventify/api/internal/shared/auth"
	"eventify/api/internal/transport/http/middleware"
	v1admin "eventify/api/internal/transport/http/v1/admin"
	v1auth "eventify/api/internal/transport/http/v1/auth"
	v1events "eventify/api/internal/transport/http/v1/events"
	v1password "eventify/api/internal/transport/http/v1/password"
	v2events "eventify/api/internal/transport/http/v2/events"
	"eventify/platform/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewApp builds the Fiber app with every route mounted.
//
// Handlers are constructed once, here, and shared by the versions that expose
// them: v1 and v2 both receive the same UpdateEventHandler value. That is the
// concrete form of "one use case, many transports".
func NewApp(pool *pgxpool.Pool, jwtProvider auth.IJWTProvider, adapter telemetry.ITelemetryAdapter) *fiber.App {
	app := fiber.New()
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.Telemetry(adapter))

	// Feature handlers. CreateEventHandler takes the pool because it opens its
	// own transaction to write the event and its outbox row atomically.
	create := events.NewCreateEventHandler(pool)
	update := events.NewUpdateEventHandler(pool)
	get := events.NewGetEventHandler(pool)
	list := events.NewGetEventsHandler(pool)
	del := events.NewDeleteEventHandler(pool)

	v1events.New(v1events.Handlers{
		Create: create, Update: update, Get: get, List: list, Delete: del,
	}).Register(app, jwtProvider)

	// Same update, get and list handler values as v1.
	v2events.New(v2events.Handlers{
		Update: update, Get: get, List: list,
	}).Register(app, jwtProvider)

	v1auth.New(v1auth.Handlers{
		Login:       users.NewLoginHandler(pool, jwtProvider),
		Signup:      users.NewSignupHandler(pool),
		GetUser:     users.NewGetUserHandler(pool),
		Permissions: users.NewGetUserPermissionsHandler(pool),
	}, jwtProvider).Register(app)

	v1password.New(v1password.Handlers{
		ChangePassword: users.NewChangePasswordHandler(pool),
	}, jwtProvider).Register(app)

	v1admin.New(v1admin.Handlers{
		ListRoles:        roles.NewListRolesHandler(pool),
		CreateRole:       roles.NewCreateRoleHandler(pool),
		AssignRole:       roles.NewAssignRoleHandler(pool),
		RemoveRole:       roles.NewRemoveRoleHandler(pool),
		GetUserRoles:     roles.NewGetUserRolesHandler(pool),
		ListPermissions:  permissions.NewListPermissionsHandler(pool),
		RolePermissions:  permissions.NewGetRolePermissionsHandler(pool),
		AssignPermission: permissions.NewAssignPermissionHandler(pool),
		RemovePermission: permissions.NewRemovePermissionHandler(pool),
	}).Register(app, jwtProvider)

	return app
}
