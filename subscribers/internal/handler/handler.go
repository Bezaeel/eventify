// Package handler defines the contract every event consumer implements, and a
// registry that maps an event name to the consumer for it.
package handler

import (
	"context"
	"fmt"

	"eventify/events"
)

// Handler consumes one event.
//
// Handle receives the raw payload bytes rather than a decoded struct: the
// registry cannot know the concrete type, and decoding belongs to the handler
// anyway, since it is the only code that knows which contract the name implies.
type Handler interface {
	Name() string
	Handle(ctx context.Context, payload []byte) error
}

// Registry resolves an event name to its Handler.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry builds a Registry from the given handlers, rejecting duplicates.
func NewRegistry(hs ...Handler) (*Registry, error) {
	r := &Registry{handlers: make(map[string]Handler, len(hs))}
	for _, h := range hs {
		if _, dup := r.handlers[h.Name()]; dup {
			return nil, fmt.Errorf("duplicate handler registered for %s", h.Name())
		}
		r.handlers[h.Name()] = h
	}
	return r, nil
}

// Names lists every event this subscriber consumes.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

// RoutingKeys lists every key the subscriber must bind its queue to.
func (r *Registry) RoutingKeys() []string {
	keys := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		keys = append(keys, events.RoutingKey(name))
	}
	return keys
}

// Dispatch routes a payload to the handler registered for name.
//
// An unknown name is an error, not a silent drop: it means a producer shipped
// an event this binary was never taught to consume, and the message should be
// nacked so it lands in the dead-letter queue rather than disappearing.
func (r *Registry) Dispatch(ctx context.Context, name string, payload []byte) error {
	h, ok := r.handlers[name]
	if !ok {
		return fmt.Errorf("no handler registered for %s", name)
	}
	return h.Handle(ctx, payload)
}
