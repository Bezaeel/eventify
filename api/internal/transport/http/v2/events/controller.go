// Package events is the HTTP v2 adapter for the event use cases.
//
// v2 differs from v1 only in its wire contract:
//
//   - `organizer` is spelled `organiser`
//   - the update response embeds the full event rather than just an id
//
// Neither change reaches internal/features. Both versions call the identical
// UpdateEventHandler with the identical UpdateEventCommand, so there is exactly
// one UPDATE statement in the codebase and exactly one place to test it.
//
// Had the behaviour changed — different rows, different rules, an extra write —
// the correct move would have been a new command type and a new handler file
// (update_event_v2.go), not a numbered method on something both versions share.
// See .claude/skills/add-endpoint-version.
package events

import (
	"eventify/api/internal/features/events"
	"eventify/api/internal/shared/auth"
	"eventify/api/internal/shared/constants"
	"eventify/api/internal/transport/http/middleware"

	"github.com/gofiber/fiber/v2"
)

// Handlers are the use cases this controller exposes. Same types as v1.
type Handlers struct {
	Update events.UpdateEventHandler
	Get    events.GetEventHandler
	List   events.GetEventsHandler
}

// Controller adapts HTTP v2 onto Handlers.
type Controller struct {
	h Handlers
}

// New builds the controller.
func New(h Handlers) *Controller { return &Controller{h: h} }

// Register mounts the v2 event routes.
func (c *Controller) Register(app *fiber.App, jwtProvider auth.IJWTProvider) {
	r := app.Group("/api/v2/events", middleware.JWT(jwtProvider))

	r.Get("/", c.List)
	r.Put("/:id", middleware.HasPermission(constants.Permissions.EventPermissions.Update), c.Update)
}
