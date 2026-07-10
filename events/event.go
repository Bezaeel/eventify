// Package events holds the wire contracts exchanged between the api module
// (which publishes via the outbox) and the subscribers module (which consumes).
//
// # Versioning
//
// Events are not versioned in their type. The pipeline is versioned instead:
// producer and consumer deploy together, and a shape change ships on both sides
// at once.
//
// That trade is deliberate, and it has one edge the pipeline cannot cover.
// During a rollout, messages published by the old producer are still sitting in
// RabbitMQ when the new consumer starts reading. Whatever is in flight will be
// decoded by the new struct. So:
//
//	Field changes must be additive. Never rename a field, never change its type,
//	never remove one that a message in flight might carry.
//
// Adding a field is safe: an old message simply leaves it zero. Renaming one
// silently zeroes it on every message already queued, and nothing fails loudly.
package events

// RoutingKey is the AMQP routing key for a given event name, e.g.
// "eventify.events.EventCreated". The relay publishes under it and subscribers
// bind their queue to it, both through this function, so the two cannot drift
// apart.
func RoutingKey(name string) string { return "eventify.events." + name }
