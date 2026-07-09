// Package events holds the wire contracts exchanged between the api module
// (which publishes via the outbox) and the subscribers module (which consumes).
//
// # Versioning
//
// A published event is a contract with other processes. A subscriber running
// last week's binary will still receive events produced by today's. Therefore:
//
//	Never modify a published event struct in place.
//
// To evolve one, add a new version package alongside the old (v1 -> v2),
// dual-publish both until every consumer has migrated, then delete v1. Adding
// a field to v1 silently breaks every consumer that validates payloads; the
// compiler will not warn you, because the break happens over the wire.
//
// Each version package declares its own Name and Version constants. The outbox
// relay joins them into a routing key; subscribers register per (name, version).
package events

// Envelope is the transport-neutral shape the relay publishes and subscribers
// decode. Payload holds the versioned event struct, marshalled to JSON.
type Envelope struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	MessageID  string `json:"message_id"`
	OccurredAt string `json:"occurred_at"`
	Payload    []byte `json:"payload"`
}

// RoutingKey is the AMQP routing key for a given event name and version, e.g.
// "eventify.events.EventCreated.v1". Both the relay and subscribers derive the
// key through this function so the two can never drift apart.
func RoutingKey(name, version string) string {
	return "eventify.events." + name + "." + version
}
