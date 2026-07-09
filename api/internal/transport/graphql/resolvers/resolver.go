package resolvers

// This file will not be regenerated automatically.
// It is the dependency-injection seam for the GraphQL server.

import (
	"context"

	"eventify/api/internal/domain"
	"eventify/api/internal/features/events"
)

// Resolver holds the use cases the GraphQL schema exposes.
//
// They are function values, not services — the very same handler methods the
// HTTP and gRPC adapters are wired with. GraphQL is one more edge over one
// implementation, and a test can inject stubs here.
type Resolver struct {
	Create func(context.Context, events.CreateEventCommand) (events.CreateEventResult, error)
	Update func(context.Context, events.UpdateEventCommand) (events.UpdateEventResult, error)
	Get    func(context.Context, events.GetEventQuery) (domain.Event, error)
	List   func(context.Context, events.GetEventsQuery) (events.GetEventsResult, error)
	Delete func(context.Context, events.DeleteEventCommand) error
}
