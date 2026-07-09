package resolvers

// This file will not be regenerated automatically.
// It is the dependency-injection seam for the GraphQL server.

import (
	"eventify/api/internal/features/events"
)

// Resolver holds the use cases the GraphQL schema exposes.
//
// It holds feature handlers, not a service — the very same CreateEventHandler
// value the HTTP and gRPC adapters hold. GraphQL is one more edge over one
// implementation.
type Resolver struct {
	Create events.CreateEventHandler
	Update events.UpdateEventHandler
	Get    events.GetEventHandler
	List   events.GetEventsHandler
	Delete events.DeleteEventHandler
}

// NewResolver builds the root resolver.
func NewResolver(
	create events.CreateEventHandler,
	update events.UpdateEventHandler,
	get events.GetEventHandler,
	list events.GetEventsHandler,
	del events.DeleteEventHandler,
) *Resolver {
	return &Resolver{Create: create, Update: update, Get: get, List: list, Delete: del}
}
