package handler_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"eventify/events"
	"eventify/platform/logger"
	"eventify/subscribers/internal/handler"
	"eventify/subscribers/tests/integration/testsupport"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// payloadFor marshals an event the way the relay publishes it: the bare struct,
// no envelope around it.
func payloadFor(t *testing.T, evt events.EventCreated) []byte {
	t.Helper()
	body, err := json.Marshal(evt)
	require.NoError(t, err)
	return body
}

func sampleEvent(messageID uuid.UUID) events.EventCreated {
	return events.EventCreated{
		MessageID:  messageID,
		ID:         uuid.New(),
		Name:       "Tech Conference",
		Type:       "conference",
		DoneBy:     uuid.NewString(),
		OccurredAt: time.Now().UTC().Truncate(time.Second),
	}
}

func countFor(t *testing.T, pool *pgxpool.Pool, messageID uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM analytics_events WHERE message_id = $1`, messageID).Scan(&n))
	return n
}

// The relay guarantees at-least-once delivery: it can publish a row and crash
// before marking it completed, redelivering on restart. A consumer that is not
// idempotent double-counts in production and passes every test that delivers
// each message exactly once. This is that test.
func TestIntegrationEventCreated_IsIdempotentOnMessageID(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	h := handler.NewEventCreated(pool, logger.New(false))
	messageID := uuid.New()
	body := payloadFor(t, sampleEvent(messageID))

	require.NoError(t, h.Handle(ctx, body))
	require.NoError(t, h.Handle(ctx, body), "redelivery must not error")

	require.Equal(t, 1, countFor(t, pool, messageID), "redelivery must not insert a second row")
}

func TestIntegrationEventCreated_PersistsThePayload(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	h := handler.NewEventCreated(pool, logger.New(false))
	messageID := uuid.New()
	evt := sampleEvent(messageID)

	require.NoError(t, h.Handle(ctx, payloadFor(t, evt)))

	var (
		eventID                 uuid.UUID
		name, evType, createdBy string
		occurredAt              time.Time
	)
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT event_id, name, type, created_by, occurred_at
		   FROM analytics_events WHERE message_id = $1`, messageID).
		Scan(&eventID, &name, &evType, &createdBy, &occurredAt))

	require.Equal(t, evt.ID, eventID)
	require.Equal(t, evt.Name, name)
	require.Equal(t, evt.Type, evType)
	require.Equal(t, evt.DoneBy, createdBy)
	require.WithinDuration(t, evt.OccurredAt, occurredAt.UTC(), time.Second)
}

func TestIntegrationEventCreated_RejectsAMalformedPayload(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	h := handler.NewEventCreated(pool, logger.New(false))

	// It must return an error so the subscriber nacks. The old consumer logged
	// the failure and acked anyway, silently discarding the event.
	require.Error(t, h.Handle(ctx, []byte(`{"not":`)))
}

// Deduplication is only as good as the key. A payload with no message_id would
// insert under the nil UUID, so the second such event would be silently
// swallowed as a duplicate of the first.
func TestIntegrationEventCreated_RejectsAPayloadWithoutAMessageID(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	h := handler.NewEventCreated(pool, logger.New(false))
	evt := sampleEvent(uuid.Nil)

	err := h.Handle(ctx, payloadFor(t, evt))
	require.Error(t, err)
	require.Contains(t, err.Error(), "message_id")
}

// Dispatch resolves an event name to a handler. An unknown name must be an
// error, not a silent drop, so the message lands in the dead-letter queue.
func TestRegistry_DispatchRejectsAnUnknownEvent(t *testing.T) {
	r, err := handler.NewRegistry()
	require.NoError(t, err)

	err = r.Dispatch(context.Background(), "Ghost", []byte(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no handler registered")
}

func TestRegistry_RejectsDuplicateHandlers(t *testing.T) {
	h := handler.NewEventCreated(nil, logger.New(false))

	_, err := handler.NewRegistry(h, h)
	require.Error(t, err, "two handlers for one event name is a wiring bug")
	require.Contains(t, err.Error(), "duplicate handler")
}

func TestRegistry_RoutingKeysCoverEveryHandler(t *testing.T) {
	r, err := handler.NewRegistry(handler.NewEventCreated(nil, logger.New(false)))
	require.NoError(t, err)

	require.Equal(t, []string{events.RoutingKey(events.EventCreatedName)}, r.RoutingKeys())
	require.Equal(t, []string{events.EventCreatedName}, r.Names())
}
