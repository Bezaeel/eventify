// Package v1 is the first published version of the EventCreated contract.
//
// Frozen. See the events package doc: to change this shape, add a v2 package
// rather than editing this file.
package v1

import "time"

const (
	// Name identifies the event independently of its version.
	Name = "EventCreated"
	// Version is this package's contract version.
	Version = "v1"
)

// EventCreated is emitted when an event is created via any transport.
//
// This struct replaces two near-identical copies that used to live in the
// analytics module: events.EventCreated and domain.EventCreated, which differed
// only by an extra Id field. Both publisher and consumer now compile against
// this one declaration, so a change to it cannot desynchronise them.
//
// Those copies carried a CountryCode field. It is deliberately absent here:
// domain.Event has no country, and the only code that ever populated it was
// analytics/publisher/main.go, a synthetic load generator that picked at random
// from {NG, UK, GH, BR}. A contract field no producer can fill is worse than an
// absent one — every consumer would have to treat it as optional anyway. If
// events gain a real country, add it in a v2.
type EventCreated struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	CreatedBy  string    `json:"created_by"`
}
