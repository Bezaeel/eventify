// Package events is the HTTP v1 adapter for the event use cases.
//
// It decodes requests into commands, calls a handler, and encodes the result.
// It contains no SQL and no business rules. Compare v2, which serves different
// DTOs over the very same handlers.
package events

import (
	"eventify/api/internal/features/events"
	"eventify/api/internal/shared/auth"
	"eventify/api/internal/shared/constants"
	"eventify/api/internal/transport/http/middleware"

	"github.com/gofiber/fiber/v2"
)

// Handlers are the use cases this controller exposes.
type Handlers struct {
	Create events.CreateEventHandler
	Update events.UpdateEventHandler
	Get    events.GetEventHandler
	List   events.GetEventsHandler
	Delete events.DeleteEventHandler
}

// Controller adapts HTTP v1 onto Handlers.
type Controller struct {
	h Handlers
}

// New builds the controller.
func New(h Handlers) *Controller { return &Controller{h: h} }

// Register mounts the v1 event routes.
//
// Every permission-guarded route mounts JWT first. The v2 update route used to
// mount HasPermission alone, so claims were never populated and it answered 401
// unconditionally.
func (c *Controller) Register(app *fiber.App, jwtProvider auth.IJWTProvider) {
	r := app.Group("/api/v1/events", middleware.JWT(jwtProvider))

	r.Get("/", c.List)
	r.Get("/:id", c.Get)
	r.Post("/", middleware.HasPermission(constants.Permissions.EventPermissions.Create), c.Create)
	r.Put("/:id", middleware.HasPermission(constants.Permissions.EventPermissions.Update), c.Update)
	r.Delete("/:id", middleware.HasPermission(constants.Permissions.EventPermissions.Delete), c.Delete)
}
