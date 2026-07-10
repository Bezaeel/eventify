package events

import (
	"time"

	"github.com/google/uuid"
)

// EventCreatedName identifies the EventCreated contract on the wire and in the
// outbox_messages.name column.
//
// It lives here, next to the struct it names, because the producer (api, at
// Enqueue) and the consumer (the relay's processor, at CanProcess) run in
// different binaries and would otherwise each carry their own string literal. A
// typo in one would enqueue rows that no processor claims; they would exhaust
// their retries and land in the poison state, and no compiler would have
// objected.
const EventCreatedName = "EventCreated"

// EventCreated is published when an event is created via any transport.
//
// MessageID is the deduplication key. The outbox guarantees at-least-once
// delivery — the relay may publish a row and die before recording that it did —
// so every consumer must tolerate seeing the same MessageID twice.
//
// ID is the created event's own identifier, distinct from MessageID: two
// deliveries of one creation share an ID but not a MessageID.
type EventCreated struct {
	OccurredAt time.Time `json:"occurred_at"`
	MessageID  uuid.UUID `json:"message_id"`
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	DoneBy     string    `json:"done_by"`
}
