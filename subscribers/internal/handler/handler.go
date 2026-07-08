// Package handler defines the contract every event consumer implements, and a
// registry that maps (event name, version) to the consumer for it.
package handler

import (
	"context"
	"fmt"

	"eventify/events"
)

// Handler consumes one version of one event.
//
// One Handler per (Name, Version) pair. When an event gains a v2, you register
// a second Handler rather than teaching the v1 handler to branch on version —
// that branch is how a consumer ends up silently mishandling old payloads.
type Handler interface {
	Name() string
	Version() string
	Handle(ctx context.Context, env events.Envelope) error
}

// Registry resolves an envelope to its Handler.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry builds a Registry from the given handlers, rejecting duplicates.
func NewRegistry(hs ...Handler) (*Registry, error) {
	r := &Registry{handlers: make(map[string]Handler, len(hs))}
	for _, h := range hs {
		key := events.RoutingKey(h.Name(), h.Version())
		if _, dup := r.handlers[key]; dup {
			return nil, fmt.Errorf("duplicate handler registered for %s", key)
		}
		r.handlers[key] = h
	}
	return r, nil
}

// RoutingKeys lists every key the subscriber must bind its queue to.
func (r *Registry) RoutingKeys() []string {
	keys := make([]string, 0, len(r.handlers))
	for k := range r.handlers {
		keys = append(keys, k)
	}
	return keys
}

// Dispatch routes env to its Handler.
//
// An unknown (name, version) is an error, not a silent drop: it means a
// producer shipped an event this binary was never taught to consume, and the
// message should be nacked so it lands in the dead-letter queue rather than
// disappearing.
func (r *Registry) Dispatch(ctx context.Context, env events.Envelope) error {
	key := events.RoutingKey(env.Name, env.Version)
	h, ok := r.handlers[key]
	if !ok {
		return fmt.Errorf("no handler registered for %s", key)
	}
	return h.Handle(ctx, env)
}
