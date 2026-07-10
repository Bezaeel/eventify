package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"eventify/events"
	"eventify/platform/logger"
	"eventify/platform/postgres"

	"github.com/google/uuid"
)

// EventCreated projects the EventCreated event into the analytics read model.
type EventCreated struct {
	db  postgres.Querier
	log *logger.Logger
}

// NewEventCreated builds the handler.
func NewEventCreated(db postgres.Querier, log *logger.Logger) *EventCreated {
	return &EventCreated{db: db, log: log}
}

// Name is the event this handler consumes.
func (h *EventCreated) Name() string { return events.EventCreatedName }

// Handle persists the event idempotently.
//
// The outbox relay guarantees at-least-once delivery, so this method must
// tolerate seeing the same MessageID twice. ON CONFLICT DO NOTHING makes the
// second delivery a no-op rather than a duplicate analytics row.
//
// MessageID is read from the payload, not from broker metadata. The two carry
// the same value today, but only the payload survives a replay from a dump or a
// hop through a bridge that does not preserve message IDs — and a lost
// deduplication key silently turns at-least-once into duplicate rows.
func (h *EventCreated) Handle(ctx context.Context, payload []byte) error {
	var evt events.EventCreated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", events.EventCreatedName, err)
	}
	if evt.MessageID == uuid.Nil {
		return fmt.Errorf("%s payload carries no message_id; cannot deduplicate", events.EventCreatedName)
	}

	_, err := h.db.Exec(ctx,
		`INSERT INTO analytics_events
		     (message_id, event_id, name, type, created_by, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (message_id) DO NOTHING`,
		evt.MessageID, evt.ID, evt.Name, evt.Type, evt.DoneBy, evt.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("persist analytics event %s: %w", evt.MessageID, err)
	}

	h.log.WithFields(logger.Fields{
		"message_id": evt.MessageID,
		"event_name": evt.Name,
		"done_by":    evt.DoneBy,
	}).Info("projected EventCreated")

	return nil
}
