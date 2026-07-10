// Package processors turns a claimed outbox row into a published message.
//
// The relay does not know how to publish any particular event. It knows how to
// claim rows and record outcomes; it asks a processor to do the rest.
//
// Two kinds exist, and both satisfy IOutboxProcessor, so the relay's loop cannot
// tell them apart:
//
//   - Generic publishes the stored payload unchanged. Most events want this. It
//     needs no type parameter, because it never looks inside the bytes.
//   - Base[T] decodes the payload into T and hands it to a process function, for
//     an event that must enrich it, call another service, or write a second row
//     before it is safe to publish.
package processors

import (
	"context"
	"encoding/json"
	"fmt"

	"eventify/events"
	"eventify/outbox"

	"github.com/google/uuid"
)

// Publisher is the bus a processor writes to. Declared here rather than
// imported from watermill so processors can be tested against an in-memory fake.
type Publisher interface {
	Publish(ctx context.Context, routingKey, messageID string, body []byte) error
}

// IOutboxProcessor handles one payload type.
//
// CanProcess must be cheap and side-effect free: the relay calls it on every
// registered processor for every claimed message until one claims it.
type IOutboxProcessor interface {
	CanProcess(m *outbox.Message) bool
	ProcessAsync(ctx context.Context, m *outbox.Message) error
}

// publish sends a message's stored bytes under its payload type's routing key.
//
// The bytes go out exactly as they were enqueued, rather than being re-encoded
// from a struct. Round-tripping through the current binary's view of the
// contract would silently drop any field it does not know about, which is the
// one thing an additive-only contract is supposed to survive.
func publish(ctx context.Context, pub Publisher, m *outbox.Message) error {
	return pub.Publish(ctx, events.RoutingKey(m.PayloadType), m.MessageID.String(), m.Payload)
}

// Generic publishes an event's payload unchanged.
//
// It does not decode the payload, and so cannot reject a malformed one. That is
// deliberate: the relay is a pipe. A payload the consumer cannot read is the
// consumer's error to raise, where a dead-letter queue can hold it — not the
// relay's, where it would sit as an EXCEEDED row nobody is watching.
type Generic struct {
	pub         Publisher
	payloadType string
}

// NewGeneric builds a passthrough processor for the given payload type.
func NewGeneric(pub Publisher, payloadType string) *Generic {
	return &Generic{pub: pub, payloadType: payloadType}
}

// CanProcess reports whether m carries the payload type this processor handles.
func (g *Generic) CanProcess(m *outbox.Message) bool { return m.PayloadType == g.payloadType }

// ProcessAsync publishes the payload as stored.
func (g *Generic) ProcessAsync(ctx context.Context, m *outbox.Message) error {
	return publish(ctx, g.pub, m)
}

// Base is the extension point for an event that needs work done before it is
// published. Embed it, and call Process from ProcessAsync.
//
// It dispatches on the declared PayloadType, never on reflect.TypeOf(T). A Go
// type name is not a stable identifier: the row was written by a different
// binary, possibly before a refactor renamed the struct or moved its package.
// Rows already queued would stop matching, and the relay would poison a backlog
// it was perfectly capable of publishing.
type Base[T any] struct {
	Pub         Publisher
	PayloadType string
}

// CanProcess reports whether m carries the payload type this processor handles.
func (b *Base[T]) CanProcess(m *outbox.Message) bool { return m.PayloadType == b.PayloadType }

// Process decodes m's payload into T and calls fn with it.
//
// A payload that will not decode is returned as an error, not logged and
// swallowed. The relay counts the attempt, and after MaxAttempts the message is
// marked EXCEEDED rather than retried forever — a malformed payload will not
// become well-formed on the eleventh try, but it will be visible in the table.
func (b *Base[T]) Process(ctx context.Context, m *outbox.Message,
	fn func(ctx context.Context, messageID uuid.UUID, payload T) error) error {

	var payload T
	if err := json.Unmarshal(m.Payload, &payload); err != nil {
		return fmt.Errorf("decode %s payload for message %s: %w", m.PayloadType, m.MessageID, err)
	}
	return fn(ctx, m.MessageID, payload)
}

// Publish sends the message's stored bytes, for a process function that enriched
// nothing and wants the passthrough behaviour after doing its own work.
func (b *Base[T]) Publish(ctx context.Context, m *outbox.Message) error {
	return publish(ctx, b.Pub, m)
}
