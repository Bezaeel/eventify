package events_test

import (
	"context"
	"testing"
	"time"

	"eventify/api/internal/features/events"
	"eventify/api/tests/integration/testsupport"
	eventcreatedv1 "eventify/events/eventcreated/v1"
	"eventify/platform/apperrors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// seedUser inserts a user so events.created_by satisfies its foreign key.
func seedUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password, first_name, last_name, created_at)
		 VALUES ($1, $2, 'x', 'Test', 'User', now())`, id, id.String()+"@example.com")
	require.NoError(t, err)
	return id
}

func newEventCmd(createdBy uuid.UUID) events.CreateEventCommand {
	return events.CreateEventCommand{
		Name:        "Tech Conference",
		Description: "Annual conference",
		Location:    "Lagos",
		Date:        time.Now().Add(24 * time.Hour).UTC().Truncate(time.Millisecond),
		Organizer:   "Bezaeel",
		Category:    "conference",
		Tags:        []string{"go", "backend"},
		Capacity:    100,
		CreatedBy:   createdBy,
	}
}

func TestIntegrationCreateEvent(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	userID := seedUser(t, pool)
	h := events.NewCreateEventHandler(pool)

	t.Run("writes the event and its outbox row in one transaction", func(t *testing.T) {
		res, err := h.Handle(ctx, newEventCmd(userID))
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, res.EventID)

		var name string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT name FROM events WHERE id = $1`, res.EventID).Scan(&name))
		require.Equal(t, "Tech Conference", name)

		// The whole point of the outbox: the row exists because the insert
		// committed, not because a publish succeeded.
		var (
			evName, evVersion string
			publishedAt       *time.Time
		)
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT name, version, published_at FROM outbox_messages
			  WHERE payload->>'id' = $1`, res.EventID.String()).
			Scan(&evName, &evVersion, &publishedAt))

		require.Equal(t, eventcreatedv1.Name, evName)
		require.Equal(t, eventcreatedv1.Version, evVersion)
		require.Nil(t, publishedAt, "relay has not run; the row must be unpublished")
	})

	t.Run("rejects capacity below one", func(t *testing.T) {
		cmd := newEventCmd(userID)
		cmd.Capacity = 0

		_, err := h.Handle(ctx, cmd)
		require.Equal(t, apperrors.Invalid, apperrors.KindOf(err))
	})

	t.Run("a failed insert leaves no outbox row", func(t *testing.T) {
		// created_by violates the foreign key, so the INSERT fails and the
		// transaction rolls back. If Enqueue ran outside the transaction, an
		// orphan outbox row would survive and the relay would publish an event
		// for an event that does not exist.
		before := countOutbox(t, pool)

		cmd := newEventCmd(uuid.New()) // no such user
		_, err := h.Handle(ctx, cmd)
		require.Error(t, err)
		require.Equal(t, apperrors.Internal, apperrors.KindOf(err))

		require.Equal(t, before, countOutbox(t, pool))
	})
}

func countOutbox(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM outbox_messages`).Scan(&n))
	return n
}

func TestIntegrationUpdateEvent(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	userID := seedUser(t, pool)
	created, err := events.NewCreateEventHandler(pool).Handle(ctx, newEventCmd(userID))
	require.NoError(t, err)

	update := events.NewUpdateEventHandler(pool)
	get := events.NewGetEventHandler(pool)

	t.Run("updates in place and persists every field", func(t *testing.T) {
		// The old gorm mapper set no Id, so Save() INSERTed a second row with a
		// nil UUID; and it silently dropped Description, Organizer, Category,
		// Tags and Capacity. Both regressions are asserted against here.
		before := countEvents(t, pool)

		res, err := update.Handle(ctx, events.UpdateEventCommand{
			EventID:     created.EventID,
			Name:        "Renamed",
			Description: "New description",
			Location:    "Abuja",
			Date:        time.Now().Add(48 * time.Hour).UTC().Truncate(time.Millisecond),
			Organizer:   "New Organizer",
			Category:    "workshop",
			Tags:        []string{"updated"},
			Capacity:    42,
		})
		require.NoError(t, err)
		require.Equal(t, created.EventID, res.EventID, "must update in place, not insert")
		require.Equal(t, before, countEvents(t, pool), "no new row may appear")

		got, err := get.Handle(ctx, events.GetEventQuery{EventID: created.EventID})
		require.NoError(t, err)
		require.Equal(t, "Renamed", got.Name)
		require.Equal(t, "New description", got.Description)
		require.Equal(t, "New Organizer", got.Organizer)
		require.Equal(t, "workshop", got.Category)
		require.Equal(t, []string{"updated"}, got.Tags)
		require.Equal(t, 42, got.Capacity)
	})

	t.Run("unknown id is NotFound, not a silent insert", func(t *testing.T) {
		_, err := update.Handle(ctx, events.UpdateEventCommand{
			EventID: uuid.New(), Name: "ghost", Capacity: 1,
		})
		require.Equal(t, apperrors.NotFound, apperrors.KindOf(err))
	})
}

func countEvents(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM events`).Scan(&n))
	return n
}

func TestIntegrationGetEvents(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	userID := seedUser(t, pool)
	create := events.NewCreateEventHandler(pool)
	for range 5 {
		_, err := create.Handle(ctx, newEventCmd(userID))
		require.NoError(t, err)
	}

	list := events.NewGetEventsHandler(pool)

	t.Run("pages in SQL and reports the full total", func(t *testing.T) {
		res, err := list.Handle(ctx, events.GetEventsQuery{Limit: 2, Offset: 0})
		require.NoError(t, err)
		require.Len(t, res.Events, 2, "limit must reach the query")
		require.Equal(t, 5, res.Total, "total counts every row, not the page")
	})

	t.Run("caps an oversized limit", func(t *testing.T) {
		res, err := list.Handle(ctx, events.GetEventsQuery{Limit: 100000})
		require.NoError(t, err)
		require.LessOrEqual(t, len(res.Events), 200)
	})

	t.Run("round-trips json tags", func(t *testing.T) {
		res, err := list.Handle(ctx, events.GetEventsQuery{Limit: 1})
		require.NoError(t, err)
		require.Equal(t, []string{"go", "backend"}, res.Events[0].Tags)
	})
}

func TestIntegrationGetAndDeleteEvent(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	ctx := context.Background()

	userID := seedUser(t, pool)
	created, err := events.NewCreateEventHandler(pool).Handle(ctx, newEventCmd(userID))
	require.NoError(t, err)

	get := events.NewGetEventHandler(pool)
	del := events.NewDeleteEventHandler(pool)

	t.Run("get returns NotFound for an unknown id", func(t *testing.T) {
		// The old GetEventById returned nil for both "missing" and "database
		// exploded", so the caller could not tell them apart.
		_, err := get.Handle(ctx, events.GetEventQuery{EventID: uuid.New()})
		require.Equal(t, apperrors.NotFound, apperrors.KindOf(err))
	})

	t.Run("delete removes the row", func(t *testing.T) {
		require.NoError(t, del.Handle(ctx, events.DeleteEventCommand{EventID: created.EventID}))

		_, err := get.Handle(ctx, events.GetEventQuery{EventID: created.EventID})
		require.Equal(t, apperrors.NotFound, apperrors.KindOf(err))
	})

	t.Run("deleting a missing row is NotFound, not success", func(t *testing.T) {
		// The old DeleteEvent returned nil regardless, so DELETE of a
		// nonexistent id answered 200 OK.
		err := del.Handle(ctx, events.DeleteEventCommand{EventID: uuid.New()})
		require.Equal(t, apperrors.NotFound, apperrors.KindOf(err))
	})
}
