package handler_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"eventify/events"
	eventcreatedv1 "eventify/events/eventcreated/v1"
	"eventify/platform/logger"
	"eventify/subscribers/internal/handler"
	"eventify/subscribers/tests/integration/testsupport"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func envelope(t *testing.T, messageID string, payload eventcreatedv1.EventCreated) events.Envelope {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return events.Envelope{
		Name:       eventcreatedv1.Name,
		Version:    eventcreatedv1.Version,
		MessageID:  messageID,
		OccurredAt: payload.OccurredAt.Format(time.RFC3339),
		Payload:    body,
	}
}

func samplePayload() eventcreatedv1.EventCreated {
	return eventcreatedv1.EventCreated{
		ID:         uuid.NewString(),
		Name:       "Tech Conference",
		Type:       "conference",
		CreatedBy:  uuid.NewString(),
		OccurredAt: time.Now().UTC().Truncate(time.Second),
	}
}

func countFor(t *testing.T, pool *pgxpool.Pool, messageID string) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM analytics_events WHERE message_id = $1`, messageID).Scan(&n))
	return n
}

// The relay guarantees at-least-once delivery: it can publish a row and crash
// before marking it published, redelivering on restart. A consumer that is not
// idempotent double-counts in production and passes every test that delivers
// each message exactly once. This is that test.
func TestIntegrationEventCreatedV1_IsIdempotentOnMessageID(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	h := handler.NewEventCreatedV1(pool, logger.New(false))
	messageID := uuid.NewString()
	env := envelope(t, messageID, samplePayload())

	require.NoError(t, h.Handle(ctx, env))
	require.NoError(t, h.Handle(ctx, env), "redelivery must not error")

	require.Equal(t, 1, countFor(t, pool, messageID), "redelivery must not insert a second row")
}

func TestIntegrationEventCreatedV1_PersistsThePayload(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	h := handler.NewEventCreatedV1(pool, logger.New(false))
	messageID := uuid.NewString()
	payload := samplePayload()

	require.NoError(t, h.Handle(ctx, envelope(t, messageID, payload)))

	var (
		eventID, name, evType, createdBy string
		occurredAt                       time.Time
	)
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT event_id::text, name, type, created_by, occurred_at
		   FROM analytics_events WHERE message_id = $1`, messageID).
		Scan(&eventID, &name, &evType, &createdBy, &occurredAt))

	require.Equal(t, payload.ID, eventID)
	require.Equal(t, payload.Name, name)
	require.Equal(t, payload.Type, evType)
	require.Equal(t, payload.CreatedBy, createdBy)
	require.WithinDuration(t, payload.OccurredAt, occurredAt.UTC(), time.Second)
}

func TestIntegrationEventCreatedV1_RejectsAMalformedPayload(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	h := handler.NewEventCreatedV1(pool, logger.New(false))
	env := events.Envelope{
		Name: eventcreatedv1.Name, Version: eventcreatedv1.Version,
		MessageID: uuid.NewString(), Payload: []byte(`{"not":`),
	}

	// It must return an error so the subscriber nacks. The old consumer logged
	// the failure and acked anyway, silently discarding the event.
	require.Error(t, h.Handle(ctx, env))
}

// Dispatch resolves (name, version) to a handler. An unknown pair must be an
// error, not a silent drop, so the message lands in the dead-letter queue.
func TestRegistry_DispatchRejectsAnUnknownEvent(t *testing.T) {
	r, err := handler.NewRegistry()
	require.NoError(t, err)

	err = r.Dispatch(context.Background(), events.Envelope{Name: "Ghost", Version: "v9"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no handler registered")
}

func TestRegistry_RejectsDuplicateHandlers(t *testing.T) {
	h := handler.NewEventCreatedV1(nil, logger.New(false))

	_, err := handler.NewRegistry(h, h)
	require.Error(t, err, "two handlers for one (name, version) is a wiring bug")
	require.Contains(t, err.Error(), "duplicate handler")
}

func TestRegistry_RoutingKeysCoverEveryHandler(t *testing.T) {
	r, err := handler.NewRegistry(handler.NewEventCreatedV1(nil, logger.New(false)))
	require.NoError(t, err)

	require.Equal(t,
		[]string{events.RoutingKey(eventcreatedv1.Name, eventcreatedv1.Version)},
		r.RoutingKeys())
}
