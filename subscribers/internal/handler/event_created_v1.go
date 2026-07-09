package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"eventify/events"
	eventcreatedv1 "eventify/events/eventcreated/v1"
	"eventify/platform/logger"
	"eventify/platform/postgres"
)

// EventCreatedV1 projects EventCreated.v1 into the analytics read model.
//
// This replaces the previous implementation, which POSTed each event to a
// hardcoded Druid firehose URL carrying a task ID captured in July 2025
// ("index_events_cebopfkn_2025-07-19T…"). That task no longer exists, so every
// send failed — and the failure was logged, not returned, so the message was
// acked and dropped regardless.
type EventCreatedV1 struct {
	db  postgres.Querier
	log *logger.Logger
}

// NewEventCreatedV1 builds the handler.
func NewEventCreatedV1(db postgres.Querier, log *logger.Logger) *EventCreatedV1 {
	return &EventCreatedV1{db: db, log: log}
}

func (h *EventCreatedV1) Name() string    { return eventcreatedv1.Name }
func (h *EventCreatedV1) Version() string { return eventcreatedv1.Version }

// Handle persists the event idempotently.
//
// The outbox relay guarantees at-least-once delivery, so this method must
// tolerate seeing the same MessageID twice. ON CONFLICT DO NOTHING makes the
// second delivery a no-op rather than a duplicate analytics row.
func (h *EventCreatedV1) Handle(ctx context.Context, env events.Envelope) error {
	var payload eventcreatedv1.EventCreated
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", env.Name, err)
	}

	_, err := h.db.Exec(ctx,
		`INSERT INTO analytics_events
		     (message_id, event_id, name, type, created_by, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (message_id) DO NOTHING`,
		env.MessageID, payload.ID, payload.Name, payload.Type,
		payload.CreatedBy, payload.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("persist analytics event %s: %w", env.MessageID, err)
	}

	h.log.WithFields(logger.Fields{
		"message_id": env.MessageID,
		"event_name": payload.Name,
		"created_by": payload.CreatedBy,
	}).Info("projected EventCreated.v1")

	return nil
}
